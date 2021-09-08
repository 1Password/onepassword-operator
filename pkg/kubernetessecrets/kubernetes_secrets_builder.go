package kubernetessecrets

import (
	"context"
	"fmt"

	"regexp"
	"strings"

	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/1Password/onepassword-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeValidate "k8s.io/apimachinery/pkg/util/validation"

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

func CreateKubernetesSecretFromItem(kubeClient kubernetesClient.Client, secretName, namespace string, item *onepassword.Item, autoRestart string) error {

	itemVersion := fmt.Sprint(item.Version)
	annotations := map[string]string{
		VersionAnnotation:  itemVersion,
		ItemPathAnnotation: fmt.Sprintf("vaults/%v/items/%v", item.Vault.ID, item.ID),
	}
	if autoRestart != "" {
		_, err := utils.StringToBool(autoRestart)
		if err != nil {
			log.Error(err, "Error parsing %v annotation on Secret %v. Must be true or false. Defaulting to false.", RestartDeploymentsAnnotation, secretName)
			return err
		}
		annotations[RestartDeploymentsAnnotation] = autoRestart
	}
	secret := BuildKubernetesSecretFromOnePasswordItem(secretName, namespace, annotations, *item)

	currentSecret := &corev1.Secret{}
	err := kubeClient.Get(context.Background(), types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, currentSecret)
	if err != nil && errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Creating Secret %v at namespace '%v'", secret.Name, secret.Namespace))
		return kubeClient.Create(context.Background(), secret)
	} else if err != nil {
		return err
	}

	if currentSecret.Annotations[VersionAnnotation] != itemVersion {
		log.Info(fmt.Sprintf("Updating Secret %v at namespace '%v'", secret.Name, secret.Namespace))
		currentSecret.ObjectMeta.Annotations = annotations
		currentSecret.Data = secret.Data
		return kubeClient.Update(context.Background(), currentSecret)
	}

	log.Info(fmt.Sprintf("Secret with name %v and version %v already exists", secret.Name, secret.Annotations[VersionAnnotation]))
	return nil
}

func BuildKubernetesSecretFromOnePasswordItem(name, namespace string, annotations map[string]string, item onepassword.Item) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        formatSecretName(name),
			Namespace:   namespace,
			Annotations: annotations,
		},
		Data: BuildKubernetesSecretData(item.Fields),
	}
}

func BuildKubernetesSecretData(fields []*onepassword.ItemField) map[string][]byte {
	secretData := map[string][]byte{}
	for i := 0; i < len(fields); i++ {
		if fields[i].Value != "" {
			key := formatSecretName(fields[i].Label)
			secretData[key] = []byte(fields[i].Value)
		}
	}
	return secretData
}

// formatSecretName rewrites a value to be a valid Secret name or Secret data key.
//
// The Secret meta.name and data keys must be valid DNS subdomain names (https://kubernetes.io/docs/concepts/configuration/secret/#overview-of-secrets)
func formatSecretName(value string) string {
	if errs := kubeValidate.IsDNS1123Subdomain(value); len(errs) == 0 {
		return value
	}
	return createValidSecretName(value)
}

var invalidDNS1123Chars = regexp.MustCompile("[^a-zA-Z0-9-_.]+")

func createValidSecretName(value string) string {
	result := invalidDNS1123Chars.ReplaceAllString(value, "-")

	if len(result) > kubeValidate.DNS1123SubdomainMaxLength {
		result = result[0:kubeValidate.DNS1123SubdomainMaxLength]
	}

	// first and last character MUST be alphanumeric
	return strings.Trim(result, "-._")
}
