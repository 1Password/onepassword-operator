package onepassword

import (
	"context"
	"fmt"
	"time"

	kubeSecrets "github.com/1Password/onepassword-operator/pkg/kubernetessecrets"

	"github.com/1Password/connect-sdk-go/connect"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const envHostVariable = "OP_HOST"

var log = logf.Log.WithName("update_op_kubernetes_secrets_task")

func NewManager(kubernetesClient client.Client, opConnectClient connect.Client) *SecretUpdateHandler {
	return &SecretUpdateHandler{
		client:          kubernetesClient,
		opConnectClient: opConnectClient,
	}
}

type SecretUpdateHandler struct {
	client          client.Client
	opConnectClient connect.Client
}

func (h *SecretUpdateHandler) UpdateKubernetesSecretsTask() error {
	updatedKubernetesSecrets, err := h.updateKubernetesSecrets()
	if err != nil {
		return err
	}

	return h.restartDeploymentsWithUpdatedSecrets(updatedKubernetesSecrets)
}

func (h *SecretUpdateHandler) restartDeploymentsWithUpdatedSecrets(updatedSecretsByNamespace map[string]map[string]bool) error {
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

	for i := 0; i < len(deployments.Items); i++ {
		deployment := &deployments.Items[i]
		updatedSecrets := updatedSecretsByNamespace[deployment.Namespace]
		secretName := deployment.Annotations[NameAnnotation]
		log.Info(fmt.Sprintf("Looking at secret %v for deployment %v", secretName, deployment.Name))
		if isUpdatedSecret(secretName, updatedSecrets) || IsDeploymentUsingSecrets(deployment, updatedSecrets) {
			h.restartDeployment(deployment)
		} else {
			log.Info(fmt.Sprintf("Deployment '%v' is up to date", deployment.GetName()))
		}
	}
	return nil
}

func (h *SecretUpdateHandler) restartDeployment(deployment *appsv1.Deployment) {
	log.Info(fmt.Sprintf("Deployment '%v' references an updated secret. Restarting", deployment.GetName()))
	deployment.Spec.Template.Annotations = map[string]string{
		RestartAnnotation: time.Now().String(),
	}
	err := h.client.Update(context.Background(), deployment)
	if err != nil {
		log.Error(err, "Problem restarting deployment")
	}
}

func (h *SecretUpdateHandler) updateKubernetesSecrets() (map[string]map[string]bool, error) {
	secrets := &corev1.SecretList{}
	err := h.client.List(context.Background(), secrets)
	if err != nil {
		log.Error(err, "Failed to list kubernetes secrets")
		return nil, err
	}

	updatedSecrets := map[string]map[string]bool{}
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
			log.Info(fmt.Sprintf("Updating kubernetes secret '%v'", secret.GetName()))
			secret.Annotations[VersionAnnotation] = itemVersion
			updatedSecret := kubeSecrets.BuildKubernetesSecretFromOnePasswordItem(secret.Name, secret.Namespace, secret.Annotations, *item)
			h.client.Update(context.Background(), updatedSecret)
			if updatedSecrets[secret.Namespace] == nil {
				updatedSecrets[secret.Namespace] = make(map[string]bool)
			}
			updatedSecrets[secret.Namespace][secret.Name] = true
		}
	}
	return updatedSecrets, nil
}

func isUpdatedSecret(secretName string, updatedSecrets map[string]bool) bool {
	_, ok := updatedSecrets[secretName]
	if ok {
		return true
	}
	return false
}
