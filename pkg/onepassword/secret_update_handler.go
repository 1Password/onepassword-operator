package onepassword

import (
	"context"
	"fmt"
	"time"

	onepasswordv1 "github.com/1Password/onepassword-operator/api/v1"
	kubeSecrets "github.com/1Password/onepassword-operator/pkg/kubernetessecrets"
	"github.com/1Password/onepassword-operator/pkg/logs"
	opclient "github.com/1Password/onepassword-operator/pkg/onepassword/client"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
	"github.com/1Password/onepassword-operator/pkg/utils"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const envHostVariable = "OP_HOST"
const lockTag = "operator.1password.io:ignore-secret"

var log = logf.Log.WithName("update_op_kubernetes_secrets_task")

func NewManager(kubernetesClient client.Client, opClient opclient.Client, shouldAutoRestartDeploymentsGlobal bool) *SecretUpdateHandler {
	return &SecretUpdateHandler{
		client:                             kubernetesClient,
		opClient:                           opClient,
		shouldAutoRestartDeploymentsGlobal: shouldAutoRestartDeploymentsGlobal,
	}
}

type SecretUpdateHandler struct {
	client                             client.Client
	opClient                           opclient.Client
	shouldAutoRestartDeploymentsGlobal bool
}

func (h *SecretUpdateHandler) UpdateKubernetesSecretsTask(ctx context.Context) error {
	updatedKubernetesSecrets, err := h.updateKubernetesSecrets(ctx)
	if err != nil {
		return err
	}

	return h.restartDeploymentsWithUpdatedSecrets(ctx, updatedKubernetesSecrets)
}

func (h *SecretUpdateHandler) restartDeploymentsWithUpdatedSecrets(ctx context.Context, updatedSecretsByNamespace map[string]map[string]*corev1.Secret) error {
	// No secrets to update. Exit
	if len(updatedSecretsByNamespace) == 0 || updatedSecretsByNamespace == nil {
		return nil
	}

	deployments := &appsv1.DeploymentList{}
	err := h.client.List(ctx, deployments)
	if err != nil {
		log.Error(err, "Failed to list kubernetes deployments")
		return err
	}

	if len(deployments.Items) == 0 {
		return nil
	}

	setForAutoRestartByNamespaceMap, err := h.getIsSetForAutoRestartByNamespaceMap(ctx)
	if err != nil {
		return err
	}

	for i := 0; i < len(deployments.Items); i++ {
		deployment := &deployments.Items[i]
		updatedSecrets := updatedSecretsByNamespace[deployment.Namespace]

		updatedDeploymentSecrets := GetUpdatedSecretsForDeployment(deployment, updatedSecrets)
		if len(updatedDeploymentSecrets) == 0 {
			continue
		}
		for _, secret := range updatedDeploymentSecrets {
			if isSecretSetForAutoRestart(secret, deployment, setForAutoRestartByNamespaceMap) {
				h.restartDeployment(ctx, deployment)
				continue
			}
		}

		log.V(logs.DebugLevel).Info(fmt.Sprintf("Deployment %q at namespace %q is up to date", deployment.GetName(), deployment.Namespace))

	}
	return nil
}

func (h *SecretUpdateHandler) restartDeployment(ctx context.Context, deployment *appsv1.Deployment) {
	log.Info(fmt.Sprintf("Deployment %q at namespace %q references an updated secret. Restarting", deployment.GetName(), deployment.Namespace))
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations[RestartAnnotation] = time.Now().String()
	err := h.client.Update(ctx, deployment)
	if err != nil {
		log.Error(err, "Problem restarting deployment")
	}
}

func (h *SecretUpdateHandler) updateKubernetesSecrets(ctx context.Context) (map[string]map[string]*corev1.Secret, error) {
	secrets := &corev1.SecretList{}
	err := h.client.List(ctx, secrets)
	if err != nil {
		log.Error(err, "Failed to list kubernetes secrets")
		return nil, err
	}

	updatedSecrets := map[string]map[string]*corev1.Secret{}
	for i := 0; i < len(secrets.Items); i++ {
		secret := secrets.Items[i]

		itemPath := secret.Annotations[ItemPathAnnotation]
		currentVersion := secret.Annotations[VersionAnnotation]
		if len(itemPath) == 0 || len(currentVersion) == 0 {
			continue
		}

		OnePasswordItemPath := h.getPathFromOnePasswordItem(secret)

		item, err := GetOnePasswordItemByPath(ctx, h.opClient, OnePasswordItemPath)
		if err != nil {
			log.Error(err, "failed to retrieve 1Password item at path \"%s\" for secret \"%s\"", secret.Annotations[ItemPathAnnotation], secret.Name)
			continue
		}

		itemVersion := fmt.Sprint(item.Version)
		itemPathString := fmt.Sprintf("vaults/%v/items/%v", item.VaultID, item.ID)

		if currentVersion != itemVersion || secret.Annotations[ItemPathAnnotation] != itemPathString {
			if isItemLockedForForcedRestarts(item) {
				log.V(logs.DebugLevel).Info(fmt.Sprintf("Secret '%v' has been updated in 1Password but is set to be ignored. Updates to an ignored secret will not trigger an update to a kubernetes secret or a rolling restart.", secret.GetName()))
				secret.Annotations[VersionAnnotation] = itemVersion
				secret.Annotations[ItemPathAnnotation] = itemPathString
				if err := h.client.Update(ctx, &secret); err != nil {
					log.Error(err, "failed to update secret %s annotations to version %d: %s", secret.Name, itemVersion, err)
					continue
				}
				continue
			}
			log.Info(fmt.Sprintf("Updating kubernetes secret '%v'", secret.GetName()))
			secret.Annotations[VersionAnnotation] = itemVersion
			secret.Annotations[ItemPathAnnotation] = itemPathString
			secret.Data = kubeSecrets.BuildKubernetesSecretData(item.Fields, item.Files)
			log.V(logs.DebugLevel).Info(fmt.Sprintf("New secret path: %v and version: %v", secret.Annotations[ItemPathAnnotation], secret.Annotations[VersionAnnotation]))
			if err := h.client.Update(ctx, &secret); err != nil {
				log.Error(err, "failed to update secret %s to version %d: %s", secret.Name, itemVersion, err)
				continue
			}
			if updatedSecrets[secret.Namespace] == nil {
				updatedSecrets[secret.Namespace] = make(map[string]*corev1.Secret)
			}
			updatedSecrets[secret.Namespace][secret.Name] = &secret
		}
	}
	return updatedSecrets, nil
}

func isItemLockedForForcedRestarts(item *model.Item) bool {
	tags := item.Tags
	for i := 0; i < len(tags); i++ {
		if tags[i] == lockTag {
			return true
		}
	}
	return false
}

func isUpdatedSecret(secretName string, updatedSecrets map[string]*corev1.Secret) bool {
	_, ok := updatedSecrets[secretName]
	if ok {
		return true
	}
	return false
}

func (h *SecretUpdateHandler) getIsSetForAutoRestartByNamespaceMap(ctx context.Context) (map[string]bool, error) {
	namespaces := &corev1.NamespaceList{}
	err := h.client.List(ctx, namespaces)
	if err != nil {
		log.Error(err, "Failed to list kubernetes namespaces")
		return nil, err
	}

	namespacesMap := map[string]bool{}

	for _, namespace := range namespaces.Items {
		namespacesMap[namespace.Name] = h.isNamespaceSetToAutoRestart(&namespace)
	}
	return namespacesMap, nil
}

func (h *SecretUpdateHandler) getPathFromOnePasswordItem(secret corev1.Secret) string {
	onePasswordItem := &onepasswordv1.OnePasswordItem{}

	// Search for our original OnePasswordItem if it exists
	err := h.client.Get(context.TODO(), client.ObjectKey{
		Namespace: secret.Namespace,
		Name:      secret.Name}, onePasswordItem)

	if err == nil {
		return onePasswordItem.Spec.ItemPath
	}

	// If we can't find the OnePassword Item we'll just return the annotation from the secret item.
	return secret.Annotations[ItemPathAnnotation]
}

func isSecretSetForAutoRestart(secret *corev1.Secret, deployment *appsv1.Deployment, setForAutoRestartByNamespace map[string]bool) bool {
	restartDeployment := secret.Annotations[RestartDeploymentsAnnotation]
	//If annotation for auto restarts for deployment is not set. Check for the annotation on its namepsace
	if restartDeployment == "" {
		return isDeploymentSetForAutoRestart(deployment, setForAutoRestartByNamespace)
	}

	restartDeploymentBool, err := utils.StringToBool(restartDeployment)
	if err != nil {
		log.Error(err, "Error parsing %v annotation on Secret %v. Must be true or false. Defaulting to false.", RestartDeploymentsAnnotation, secret.Name)
		return false
	}
	return restartDeploymentBool
}

func isDeploymentSetForAutoRestart(deployment *appsv1.Deployment, setForAutoRestartByNamespace map[string]bool) bool {
	restartDeployment := deployment.Annotations[RestartDeploymentsAnnotation]
	//If annotation for auto restarts for deployment is not set. Check for the annotation on its namepsace
	if restartDeployment == "" {
		return setForAutoRestartByNamespace[deployment.Namespace]
	}

	restartDeploymentBool, err := utils.StringToBool(restartDeployment)
	if err != nil {
		log.Error(err, "Error parsing %v annotation on Deployment %v. Must be true or false. Defaulting to false.", RestartDeploymentsAnnotation, deployment.Name)
		return false
	}
	return restartDeploymentBool
}

func (h *SecretUpdateHandler) isNamespaceSetToAutoRestart(namespace *corev1.Namespace) bool {
	restartDeployment := namespace.Annotations[RestartDeploymentsAnnotation]
	//If annotation for auto restarts for deployment is not set. Check environment variable set on the operator
	if restartDeployment == "" {
		return h.shouldAutoRestartDeploymentsGlobal
	}

	restartDeploymentBool, err := utils.StringToBool(restartDeployment)
	if err != nil {
		log.Error(err, "Error parsing %v annotation on Namespace %v. Must be true or false. Defaulting to false.", RestartDeploymentsAnnotation, namespace.Name)
		return false
	}
	return restartDeploymentBool
}
