package onepassword

import (
	"context"
	"fmt"
	"time"

	onepasswordv1 "github.com/1Password/onepassword-operator/api/v1"
	kubeSecrets "github.com/1Password/onepassword-operator/pkg/kubernetessecrets"
	"github.com/1Password/onepassword-operator/pkg/logs"
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

func NewManager(kubernetesClient client.Client, opConnectClient connect.Client, autoRestartWorkloadsGlobally bool) *SecretUpdateHandler {
	return &SecretUpdateHandler{
		client:                       kubernetesClient,
		opConnectClient:              opConnectClient,
		autoRestartWorkloadsGlobally: autoRestartWorkloadsGlobally,
	}
}

type SecretUpdateHandler struct {
	client                       client.Client
	opConnectClient              connect.Client
	autoRestartWorkloadsGlobally bool
}

func (h *SecretUpdateHandler) UpdateKubernetesSecretsTask() error {
	updatedKubernetesSecrets, err := h.updateKubernetesSecrets()
	if err != nil {
		return err
	}

	return h.restartWorkloadsWithUpdatedSecrets(updatedKubernetesSecrets)
}

func (h *SecretUpdateHandler) restartWorkloadsWithUpdatedSecrets(updatedSecretsByNamespace map[string]map[string]*corev1.Secret) error {
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
				h.restartWorkload(deployment)
				continue
			}
		}

		log.V(logs.DebugLevel).Info(fmt.Sprintf("Deployment %q at namespace %q is up to date", deployment.GetName(), deployment.Namespace))

	}
	return nil
}

func (h *SecretUpdateHandler) restartWorkload(workload client.Object) {
	var podTemplate *corev1.PodTemplateSpec

	switch obj := workload.(type) {
	case *appsv1.Deployment:
		podTemplate = &obj.Spec.Template
	default:
		log.Info("Unsupported workload type for restart", "type", fmt.Sprintf("%T", obj))
		return
	}

	log.Info(fmt.Sprintf("%T %q in namespace %q references an updated secret. Restarting", workload, workload.GetName(), workload.GetNamespace()))

	if podTemplate.Annotations == nil {
		podTemplate.Annotations = map[string]string{}
	}
	podTemplate.Annotations[RestartAnnotation] = time.Now().Format(time.RFC3339)

	err := h.client.Update(context.Background(), workload)
	if err != nil {
		log.Error(err, "Problem restarting workload", "name", workload.GetName())
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

		OnePasswordItemPath := h.getPathFromOnePasswordItem(secret)

		item, err := GetOnePasswordItemByPath(h.opConnectClient, OnePasswordItemPath)
		if err != nil {
			log.Error(err, "failed to retrieve 1Password item at path \"%s\" for secret \"%s\"", secret.Annotations[ItemPathAnnotation], secret.Name)
			continue
		}

		itemVersion := fmt.Sprint(item.Version)
		itemPathString := fmt.Sprintf("vaults/%v/items/%v", item.Vault.ID, item.ID)

		if currentVersion != itemVersion || secret.Annotations[ItemPathAnnotation] != itemPathString {
			if isItemLockedForForcedRestarts(item) {
				log.V(logs.DebugLevel).Info(fmt.Sprintf("Secret '%v' has been updated in 1Password but is set to be ignored. Updates to an ignored secret will not trigger an update to a kubernetes secret or a rolling restart.", secret.GetName()))
				secret.Annotations[VersionAnnotation] = itemVersion
				secret.Annotations[ItemPathAnnotation] = itemPathString
				if err := h.client.Update(context.Background(), &secret); err != nil {
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
			if err := h.client.Update(context.Background(), &secret); err != nil {
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

func isSecretSetForAutoRestart(secret *corev1.Secret, workload client.Object, setForAutoRestartByNamespace map[string]bool) bool {
	restartAnnotation := secret.Annotations[AutoRestartWorkloadAnnotation]
	if restartAnnotation == "" {
		return isWorkloadSetForAutoRestart(workload, setForAutoRestartByNamespace)
	}

	restartBool, err := utils.StringToBool(restartAnnotation)
	if err != nil {
		log.Error(err, "Error parsing %v annotation on Secret %v. Must be true or false. Defaulting to false.", AutoRestartWorkloadAnnotation, secret.Name)
		return false
	}

	return restartBool
}

func isWorkloadSetForAutoRestart(obj client.Object, setForAutoRestartByNamespace map[string]bool) bool {
	annotations := obj.GetAnnotations()
	namespace := obj.GetNamespace()
	name := obj.GetName()

	restartAnnotation := annotations[AutoRestartWorkloadAnnotation]
	if restartAnnotation == "" {
		return setForAutoRestartByNamespace[namespace]
	}

	restartBool, err := utils.StringToBool(restartAnnotation)
	if err != nil {
		log.Error(err, "Error parsing %v annotation on %T %v. Must be true or false. Defaulting to false.", AutoRestartWorkloadAnnotation, obj, name)
		return false
	}

	return restartBool
}

func (h *SecretUpdateHandler) isNamespaceSetToAutoRestart(namespace *corev1.Namespace) bool {
	restartWorkload := namespace.Annotations[AutoRestartWorkloadAnnotation]
	//If annotation for auto restarts for deployment is not set. Check environment variable set on the operator
	if restartWorkload == "" {
		return h.autoRestartWorkloadsGlobally
	}

	restartWorkloadBool, err := utils.StringToBool(restartWorkload)
	if err != nil {
		log.Error(err, "Error parsing %v annotation on Namespace %v. Must be true or false. Defaulting to false.", AutoRestartWorkloadAnnotation, namespace.Name)
		return false
	}
	return restartWorkloadBool
}
