package kubernetessecrets

import (
	"context"
	"fmt"

	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/1Password/onepassword-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubernetesClient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const OnepasswordPrefix = "operator.1password.io"
const NameAnnotation = OnepasswordPrefix + "/item-name"
const VersionAnnotation = OnepasswordPrefix + "/item-version"
const restartAnnotation = OnepasswordPrefix + "/last-restarted"
const ItemPathAnnotation = OnepasswordPrefix + "/item-path"
const RestartDeploymentsAnnotation = OnepasswordPrefix + "/auto-restart"

var log = logf.Log

func CreateKubernetesSecretFromItem(kubeClient kubernetesClient.Client, secretName, namespace string, item *onepassword.Item, autoRestart string, labels map[string]string, secretAnnotations map[string]string) error {

	itemVersion := fmt.Sprint(item.Version)

	// If secretAnnotations is nil we create an empty map so we can later assign values for the OP Annotations in the map
	if secretAnnotations == nil {
		secretAnnotations = map[string]string{}
	}

	secretAnnotations[VersionAnnotation] = itemVersion
	secretAnnotations[ItemPathAnnotation] = fmt.Sprintf("vaults/%v/items/%v", item.Vault.ID, item.ID)

	if autoRestart != "" {
		_, err := utils.StringToBool(autoRestart)
		if err != nil {
			log.Error(err, "Error parsing %v annotation on Secret %v. Must be true or false. Defaulting to false.", RestartDeploymentsAnnotation, secretName)
			return err
		}
		secretAnnotations[RestartDeploymentsAnnotation] = autoRestart
	}
	secret := BuildKubernetesSecretFromOnePasswordItem(secretName, namespace, secretAnnotations, labels, *item)

	currentSecret := &corev1.Secret{}
	err := kubeClient.Get(context.Background(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, currentSecret)
	if err != nil && errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Creating Secret %v at namespace '%v'", secret.Name, secret.Namespace))
		return kubeClient.Create(context.Background(), secret)
	} else if err != nil {
		return err
	}

	if CompareSecretFieldsWithOnePasswordItem(currentSecret.Annotations, secretAnnotations) || CompareSecretFieldsWithOnePasswordItem(currentSecret.Labels, labels) {
		log.Info(fmt.Sprintf("Updating Secret %v at namespace '%v'", secret.Name, secret.Namespace))
		currentSecret.ObjectMeta.Annotations = secretAnnotations
		currentSecret.ObjectMeta.Labels = labels
		currentSecret.Data = secret.Data
		return kubeClient.Update(context.Background(), currentSecret)
	}

	log.Info(fmt.Sprintf("Secret with name %v and version %v already exists", secret.Name, secret.Annotations[VersionAnnotation]))
	return nil
}

func BuildKubernetesSecretFromOnePasswordItem(name, namespace string, annotations map[string]string, labels map[string]string, item onepassword.Item) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Data: BuildKubernetesSecretData(item.Fields),
	}
}

func BuildKubernetesSecretData(fields []*onepassword.ItemField) map[string][]byte {
	secretData := map[string][]byte{}
	for i := 0; i < len(fields); i++ {
		if fields[i].Value != "" {
			secretData[fields[i].Label] = []byte(fields[i].Value)
		}
	}
	return secretData
}

func CompareSecretFieldsWithOnePasswordItem(currentSecretsFields map[string]string, expectedFieldsOnSecret map[string]string) bool{
	for key, value := range expectedFieldsOnSecret {
		currentValue, exists := currentSecretsFields[key]
		if !exists || currentValue != value {
			return true
		}
	}
	return false
}