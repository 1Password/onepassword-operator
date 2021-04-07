package onepassword

import (
	"context"
	"fmt"
	"time"

	kubeSecrets "github.com/1Password/onepassword-operator/pkg/kubernetessecrets"
	"github.com/1Password/onepassword-operator/pkg/utils"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const envHostVariable = "OP_HOST"
const lockTag = "operator.1password.io:ignore-secret"

var log = logf.Log.WithName("update_op_kubernetes_secrets_task")

func NewManager(kubernetesClient client.Client, opConnectClient connect.Client, shouldAutoRestartDeploymentsGlobal bool) *SecretUpdateHandler {
	return &SecretUpdateHandler{
		client:                             kubernetesClient,
		opConnectClient:                    opConnectClient,
		shouldAutoRestartDeploymentsGlobal: shouldAutoRestartDeploymentsGlobal,
	}
}

type SecretUpdateHandler struct {
	client                             client.Client
	opConnectClient                    connect.Client
	shouldAutoRestartDeploymentsGlobal bool
}

func (h *SecretUpdateHandler) UpdateKubernetesSecretsTask() error {
	updatedKubernetesSecrets, err := h.updateKubernetesSecrets()
	if err != nil {
		return err
	}

	return h.restartDeploymentsWithUpdatedSecrets(updatedKubernetesSecrets)
}

func (h *SecretUpdateHandler) restartDeploymentsWithUpdatedSecrets(updatedSecretsByNamespace map[string]map[string]*corev1.Secret) error {
	// No secrets to update. Exit
	if len(updatedSecretsByNamespace) == 0 || updatedSecretsByNamespace == nil {
		return nil
	}

	deployments := &appsv1.DeploymentList{}
	err := h.client.List(context.Background(), deployments)
	if err != nil {
		log.Error(err, "Failed to list kubernetes deployments")
		return err
	}

	if len(deployments.Items) == 0 {
		return nil
	}

	setForAutoRestartByNamespaceMap, err := h.getIsSetForAutoRestartByNamespaceMap()
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
				h.restartDeployment(deployment)
				continue
			}
		}

		log.Info(fmt.Sprintf("Deployment %q at namespace %q is up to date", deployment.GetName(), deployment.Namespace))

	}
	return nil
}

func (h *SecretUpdateHandler) restartDeployment(deployment *appsv1.Deployment) {
	log.Info(fmt.Sprintf("Deployment %q at namespace %q references an updated secret. Restarting", deployment.GetName(), deployment.Namespace))
	deployment.Spec.Template.Annotations = map[string]string{
		RestartAnnotation: time.Now().String(),
	}
	err := h.client.Update(context.Background(), deployment)
	if err != nil {
		log.Error(err, "Problem restarting deployment")
	}
}

func (h *SecretUpdateHandler) updateKubernetesSecrets() (map[string]map[string]*corev1.Secret, error) {
	secrets := &corev1.SecretList{}
	err := h.client.List(context.Background(), secrets)
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

		item, err := GetOnePasswordItemByPath(h.opConnectClient, secret.Annotations[ItemPathAnnotation])
		if err != nil {
			return nil, fmt.Errorf("Failed to retrieve item: %v", err)
		}

		itemVersion := fmt.Sprint(item.Version)
		if currentVersion != itemVersion {
			if isItemLockedForForcedRestarts(item) {
				log.Info(fmt.Sprintf("Secret '%v' has been updated in 1Password but is set to be ignored. Updates to an ignored secret will not trigger an update to a kubernetes secret or a rolling restart.", secret.GetName()))
				secret.Annotations[VersionAnnotation] = itemVersion
				h.client.Update(context.Background(), &secret)
				continue
			}
			log.Info(fmt.Sprintf("Updating kubernetes secret '%v'", secret.GetName()))
			secret.Annotations[VersionAnnotation] = itemVersion
			updatedSecret := kubeSecrets.BuildKubernetesSecretFromOnePasswordItem(secret.Name, secret.Namespace, secret.Annotations, *item)
			h.client.Update(context.Background(), updatedSecret)
			if updatedSecrets[secret.Namespace] == nil {
				updatedSecrets[secret.Namespace] = make(map[string]*corev1.Secret)
			}
			updatedSecrets[secret.Namespace][secret.Name] = &secret
		}
	}
	return updatedSecrets, nil
}

func isItemLockedForForcedRestarts(item *onepassword.Item) bool {
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

func (h *SecretUpdateHandler) getIsSetForAutoRestartByNamespaceMap() (map[string]bool, error) {
	namespaces := &corev1.NamespaceList{}
	err := h.client.List(context.Background(), namespaces)
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
