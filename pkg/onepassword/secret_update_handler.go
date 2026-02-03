package onepassword

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"

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

// const envHostVariable = "OP_HOST"
const lockTag = "operator.1password.io:ignore-secret"

var log = logf.Log.WithName("update_op_kubernetes_secrets_task")

type SecretUpdateHandlerConfig struct {
	ShouldAutoRestartWorkloadsGlobally bool
	AllowEmptyValues                   bool
	WatchedNamespaces                  []string
}

func NewSecretUpdateHandler(
	kubernetesClient client.Client,
	apiReader client.Reader,
	opClient opclient.Client,
	config SecretUpdateHandlerConfig,
) *SecretUpdateHandler {
	return &SecretUpdateHandler{
		client:    kubernetesClient,
		apiReader: apiReader,
		opClient:  opClient,
		config:    config,
	}
}

type SecretUpdateHandler struct {
	client    client.Client
	apiReader client.Reader
	opClient  opclient.Client
	config    SecretUpdateHandlerConfig
}

func (h *SecretUpdateHandler) UpdateKubernetesSecretsTask(ctx context.Context) error {
	updatedKubernetesSecrets, err := h.updateKubernetesSecrets(ctx)
	if err != nil {
		return err
	}

	return h.restartWorkloadsWithUpdatedSecrets(ctx, updatedKubernetesSecrets)
}

func (h *SecretUpdateHandler) restartWorkloadsWithUpdatedSecrets(
	ctx context.Context,
	updatedSecretsByNamespace map[string]map[string]*corev1.Secret,
) error {
	// No secrets to update. Exit
	if len(updatedSecretsByNamespace) == 0 || updatedSecretsByNamespace == nil {
		return nil
	}

	workloadTypes := []client.ObjectList{
		&appsv1.DeploymentList{},
	}

	setForAutoRestartByNamespaceMap, err := h.getIsSetForAutoRestartByNamespaceMap(ctx)
	if err != nil {
		return err
	}

	for _, list := range workloadTypes {
		if err := h.client.List(ctx, list); err != nil {
			log.Error(err, "Failed to list workloads", "type", fmt.Sprintf("%T", list))
			return err
		}

		items, err := meta.ExtractList(list)
		if err != nil {
			log.Error(err, "Failed to extract list items", "type", fmt.Sprintf("%T", list))
			return err
		}

		for _, obj := range items {
			workload, ok := obj.(client.Object)
			if !ok {
				log.Error(fmt.Errorf("unexpected type %T", obj), "Skipping non-client.Object")
				continue
			}

			podTemplate, err := getPodTemplate(workload)
			if err != nil {
				log.Error(err, "Failed to get pod template", "workload", workload.GetName())
				continue
			}

			updatedSecrets := updatedSecretsByNamespace[workload.GetNamespace()]
			if len(updatedSecrets) == 0 {
				continue
			}

			matchedSecrets := getUpdatedSecretsForPodTemplate(workload.GetAnnotations(), podTemplate, updatedSecrets)
			if len(matchedSecrets) == 0 {
				continue
			}

			for _, secret := range matchedSecrets {
				if isSecretSetForAutoRestart(secret, workload, setForAutoRestartByNamespaceMap) {
					if err := h.restartWorkload(ctx, workload); err != nil {
						log.Error(err, "Failed to restart workload", "workload", workload.GetName(), "namespace", workload.GetNamespace())
					}
					break
				}
			}

			log.V(logs.DebugLevel).Info(
				fmt.Sprintf("%T %q at namespace %q is up to date", workload, workload.GetName(), workload.GetNamespace()),
			)
		}
	}

	return nil
}

func (h *SecretUpdateHandler) restartWorkload(ctx context.Context, workload client.Object) error {
	podTemplate, err := getPodTemplate(workload)
	if err != nil {
		log.Error(err, "Unsupported workload type for restart", "type", fmt.Sprintf("%T", workload))
		return err
	}

	log.Info(
		fmt.Sprintf(
			"%T %q in namespace %q references an updated secret. Restarting",
			workload,
			workload.GetName(),
			workload.GetNamespace(),
		),
	)

	if podTemplate.Annotations == nil {
		podTemplate.Annotations = map[string]string{}
	}
	podTemplate.Annotations[RestartAnnotation] = time.Now().Format(time.RFC3339)

	if err := h.client.Update(ctx, workload); err != nil {
		log.Error(err, "Problem restarting workload", "name", workload.GetName())
		return err
	}
	return nil
}

func (h *SecretUpdateHandler) updateKubernetesSecrets(ctx context.Context) (
	map[string]map[string]*corev1.Secret, error,
) {
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
			log.Error(err, fmt.Sprintf("failed to retrieve 1Password item at path %s for secret %s",
				secret.Annotations[ItemPathAnnotation], secret.Name,
			))
			continue
		}

		itemVersion := fmt.Sprint(item.Version)
		itemPathString := fmt.Sprintf("vaults/%v/items/%v", item.VaultID, item.ID)

		if currentVersion != itemVersion || secret.Annotations[ItemPathAnnotation] != itemPathString {
			if isItemLockedForForcedRestarts(item) {
				log.V(logs.DebugLevel).Info(fmt.Sprintf(
					"Secret '%v' has been updated in 1Password but is set to be ignored. "+
						"Updates to an ignored secret will not trigger an update to a kubernetes secret or a rolling restart.",
					secret.GetName(),
				))
				secret.Annotations[VersionAnnotation] = itemVersion
				secret.Annotations[ItemPathAnnotation] = itemPathString
				if err := h.client.Update(ctx, &secret); err != nil {
					log.Error(err, fmt.Sprintf("failed to update secret %s annotations to version %s", secret.Name, itemVersion))
					continue
				}
				continue
			}
			log.Info(fmt.Sprintf("Updating kubernetes secret '%v'", secret.GetName()))
			secret.Annotations[VersionAnnotation] = itemVersion
			secret.Annotations[ItemPathAnnotation] = itemPathString
			secret.Data = kubeSecrets.BuildKubernetesSecretData(item.Fields, item.URLs, item.Files, h.config.AllowEmptyValues)
			log.V(logs.DebugLevel).Info(fmt.Sprintf("New secret path: %v and version: %v",
				secret.Annotations[ItemPathAnnotation], secret.Annotations[VersionAnnotation],
			))
			if err := h.client.Update(ctx, &secret); err != nil {
				log.Error(err, fmt.Sprintf("failed to update secret %s to version %s", secret.Name, itemVersion))
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
	return ok
}

func (h *SecretUpdateHandler) getIsSetForAutoRestartByNamespaceMap(
	ctx context.Context,
) (map[string]bool, error) {
	namespacesMap := map[string]bool{}

	// If watched namespaces are set get the auto-restart setting for each watched namespace
	if len(h.config.WatchedNamespaces) > 0 {
		for _, namespaceName := range h.config.WatchedNamespaces {
			namespace := &corev1.Namespace{}

			// Use the API reader to avoid the cached client: the cache fills namespace data
			// via a list of namespaces which requires list permission. With RBAC that
			// only allows get on specific namespaces, that list fails. apiReader does a
			// direct get and only needs get permission.
			err := h.apiReader.Get(ctx, client.ObjectKey{Name: namespaceName}, namespace)
			if err != nil {
				return nil, err
			}

			namespacesMap[namespaceName] = h.isNamespaceSetToAutoRestart(namespace)
		}
		return namespacesMap, nil
	}

	// If watched namespaces are not set get the auto-restart setting for all namespaces
	namespaces := &corev1.NamespaceList{}
	err := h.client.List(ctx, namespaces)
	if err != nil {
		log.Error(err, "Failed to list kubernetes namespaces")
		return nil, err
	}

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

func isSecretSetForAutoRestart(
	secret *corev1.Secret,
	workload client.Object,
	setForAutoRestartByNamespace map[string]bool,
) bool {
	restartAnnotation := secret.Annotations[AutoRestartWorkloadAnnotation]
	// If annotation for auto restarts for workload is not set. Check for the annotation on its namepsace
	if restartAnnotation == "" {
		return isWorkloadSetForAutoRestart(workload, setForAutoRestartByNamespace)
	}

	restartBool, err := utils.StringToBool(restartAnnotation)
	if err != nil {
		log.Error(
			err,
			fmt.Sprintf(
				"Error parsing %s annotation on Secret %s. Must be true or false. Defaulting to false.",
				AutoRestartWorkloadAnnotation,
				secret.Name,
			),
		)
		return false
	}

	return restartBool
}

func isWorkloadSetForAutoRestart(workload client.Object, setForAutoRestartByNamespace map[string]bool) bool {
	annotations := workload.GetAnnotations()
	namespace := workload.GetNamespace()
	name := workload.GetName()

	restartAnnotation := annotations[AutoRestartWorkloadAnnotation]
	if restartAnnotation == "" {
		return setForAutoRestartByNamespace[namespace]
	}

	restartBool, err := utils.StringToBool(restartAnnotation)
	if err != nil {
		log.Error(err, fmt.Sprintf(
			"Error parsing %s annotation on %T %s. Must be true or false. Defaulting to false.",
			AutoRestartWorkloadAnnotation, workload, name,
		))
		return false
	}

	return restartBool
}

func (h *SecretUpdateHandler) isNamespaceSetToAutoRestart(namespace *corev1.Namespace) bool {
	restartWorkload := namespace.Annotations[AutoRestartWorkloadAnnotation]
	// If annotation for auto restarts for workload is not set. Check environment variable set on the operator
	if restartWorkload == "" {
		return h.config.ShouldAutoRestartWorkloadsGlobally
	}

	restartWorkloadBool, err := utils.StringToBool(restartWorkload)
	if err != nil {
		log.Error(err, fmt.Sprintf("Error parsing %s annotation on Namespace %s. Must be true or false. Defaulting to false.",
			AutoRestartWorkloadAnnotation, namespace.Name,
		))
		return false
	}
	return restartWorkloadBool
}

func getPodTemplate(obj client.Object) (*corev1.PodTemplateSpec, error) {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return &o.Spec.Template, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", obj)
	}
}

func getUpdatedSecretsForPodTemplate(
	annotations map[string]string,
	podTemplate *corev1.PodTemplateSpec,
	secrets map[string]*corev1.Secret,
) map[string]*corev1.Secret {
	if podTemplate == nil {
		return nil
	}

	allContainers := append(podTemplate.Spec.Containers, podTemplate.Spec.InitContainers...)
	updatedSecrets := map[string]*corev1.Secret{}
	AppendAnnotationUpdatedSecret(annotations, secrets, updatedSecrets)
	AppendUpdatedContainerSecrets(allContainers, secrets, updatedSecrets)
	AppendUpdatedVolumeSecrets(podTemplate.Spec.Volumes, secrets, updatedSecrets)

	return updatedSecrets
}
