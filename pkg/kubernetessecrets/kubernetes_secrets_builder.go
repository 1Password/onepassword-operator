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
	"reflect"
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

	if ! reflect.DeepEqual(currentSecret.Annotations, secretAnnotations) || ! reflect.DeepEqual(currentSecret.Labels, labels) {
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
			Name:        formatSecretName(name),
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
			key := formatSecretDataName(fields[i].Label)
			secretData[key] = []byte(fields[i].Value)
		}
	}
	return secretData
}

// formatSecretName rewrites a value to be a valid Secret name.
//
// The Secret meta.name and data keys must be valid DNS subdomain names
// (https://kubernetes.io/docs/concepts/configuration/secret/#overview-of-secrets)
func formatSecretName(value string) string {
	if errs := kubeValidate.IsDNS1123Subdomain(value); len(errs) == 0 {
		return value
	}
	return createValidSecretName(value)
}

// formatSecretDataName rewrites a value to be a valid Secret data key.
//
// The Secret data keys must consist of alphanumeric numbers, `-`, `_` or `.`
// (https://kubernetes.io/docs/concepts/configuration/secret/#overview-of-secrets)
func formatSecretDataName(value string) string {
	if errs := kubeValidate.IsConfigMapKey(value); len(errs) == 0 {
		return value
	}
	return createValidSecretDataName(value)
}

var invalidDNS1123Chars = regexp.MustCompile("[^a-z0-9-.]+")

func createValidSecretName(value string) string {
	result := strings.ToLower(value)
	result = invalidDNS1123Chars.ReplaceAllString(result, "-")

	if len(result) > kubeValidate.DNS1123SubdomainMaxLength {
		result = result[0:kubeValidate.DNS1123SubdomainMaxLength]
	}

	// first and last character MUST be alphanumeric
	return strings.Trim(result, "-.")
}

var invalidDataChars = regexp.MustCompile("[^a-zA-Z0-9-._]+")
var invalidStartEndChars = regexp.MustCompile("(^[^a-zA-Z0-9-._]+|[^a-zA-Z0-9-._]+$)")

func createValidSecretDataName(value string) string {
	result := invalidStartEndChars.ReplaceAllString(value, "")
	result = invalidDataChars.ReplaceAllString(result, "-")

	if len(result) > kubeValidate.DNS1123SubdomainMaxLength {
		result = result[0:kubeValidate.DNS1123SubdomainMaxLength]
	}

	return result
}
