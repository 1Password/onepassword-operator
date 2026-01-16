package kubernetessecrets

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
	"github.com/1Password/onepassword-operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeValidate "k8s.io/apimachinery/pkg/util/validation"

	kubernetesClient "sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const OnepasswordPrefix = "operator.1password.io"
const NameAnnotation = OnepasswordPrefix + "/item-name"
const VersionAnnotation = OnepasswordPrefix + "/item-version"
const ItemPathAnnotation = OnepasswordPrefix + "/item-path"
const RestartDeploymentsAnnotation = OnepasswordPrefix + "/auto-restart"

var ErrCannotUpdateSecretType = errors.New("cannot change secret type: secret type is immutable")

var log = logf.Log

func CreateKubernetesSecretFromItem(
	ctx context.Context,
	kubeClient kubernetesClient.Client,
	secretName, namespace string,
	item *model.Item,
	autoRestart string,
	labels map[string]string,
	secretAnnotations map[string]string,
	secretType string,
	ownerRef *metav1.OwnerReference,
) error {
	itemVersion := fmt.Sprint(item.Version)
	if secretAnnotations == nil {
		secretAnnotations = map[string]string{}
	}
	secretAnnotations[VersionAnnotation] = itemVersion
	secretAnnotations[ItemPathAnnotation] = fmt.Sprintf("vaults/%v/items/%v", item.VaultID, item.ID)

	if autoRestart != "" {
		_, err := utils.StringToBool(autoRestart)
		if err != nil {
			return fmt.Errorf("error parsing %v annotation on Secret %v. Must be true or false. Defaulting to false",
				RestartDeploymentsAnnotation, secretName,
			)
		}
		secretAnnotations[RestartDeploymentsAnnotation] = autoRestart
	}

	// "Opaque" and "" secret types are treated the same by Kubernetes.
	secret := BuildKubernetesSecretFromOnePasswordItem(secretName, namespace, secretAnnotations, labels,
		secretType, *item, ownerRef)

	currentSecret := &corev1.Secret{}
	err := kubeClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, currentSecret)
	if err != nil && apierrors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Creating Secret %v at namespace '%v'", secret.Name, secret.Namespace))
		return kubeClient.Create(ctx, secret)
	} else if err != nil {
		return err
	}

	// Check if the secret types are being changed on the update.
	// Avoid Opaque and "" are treated as different on check.
	wantSecretType := secretType
	if wantSecretType == "" {
		wantSecretType = string(corev1.SecretTypeOpaque)
	}
	currentSecretType := string(currentSecret.Type)
	if currentSecretType == "" {
		currentSecretType = string(corev1.SecretTypeOpaque)
	}
	if currentSecretType != wantSecretType {
		return ErrCannotUpdateSecretType
	}

	currentAnnotations := currentSecret.Annotations
	currentLabels := currentSecret.Labels
	if !reflect.DeepEqual(currentAnnotations, secretAnnotations) || !reflect.DeepEqual(currentLabels, labels) {
		log.Info(fmt.Sprintf("Updating Secret %v at namespace '%v'", secret.Name, secret.Namespace))
		currentSecret.Annotations = secretAnnotations
		currentSecret.Labels = labels
		currentSecret.Data = secret.Data
		if err := kubeClient.Update(ctx, currentSecret); err != nil {
			return fmt.Errorf("kubernetes secret update failed: %w", err)
		}
		return nil
	}

	log.Info(fmt.Sprintf("Secret with name %v and version %v already exists",
		secret.Name, secret.Annotations[VersionAnnotation],
	))
	return nil
}

func BuildKubernetesSecretFromOnePasswordItem(
	name, namespace string,
	annotations map[string]string,
	labels map[string]string,
	secretType string,
	item model.Item,
	ownerRef *metav1.OwnerReference,
) *corev1.Secret {
	var ownerRefs []metav1.OwnerReference
	if ownerRef != nil {
		ownerRefs = []metav1.OwnerReference{*ownerRef}
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:            formatSecretName(name),
			Namespace:       namespace,
			Annotations:     annotations,
			Labels:          labels,
			OwnerReferences: ownerRefs,
		},
		Data: BuildKubernetesSecretData(item.Fields, item.URLs, item.Files),
		Type: corev1.SecretType(secretType),
	}
}

func BuildKubernetesSecretData(fields []model.ItemField, urls []model.ItemURL, files []model.File) map[string][]byte {
	secretData := map[string][]byte{}
	for i := 0; i < len(fields); i++ {
		key := formatSecretDataName(fields[i].Label)
		secretData[key] = []byte(fields[i].Value)
	}

	for _, url := range urls {
		if !url.Primary {
			// skip non-primary URLs
			continue
		}
		key := formatSecretDataName(url.Label)
		secretData[key] = []byte(url.URL)
	}

	// populate unpopulated fields from files
	for _, file := range files {
		content, err := file.Content()
		if err != nil {
			log.Error(err, fmt.Sprintf("Could not load contents of file %s", file.Name))
			continue
		}
		if content != nil {
			key := file.Name
			if secretData[key] == nil {
				secretData[key] = content
			} else {
				log.Info(fmt.Sprintf("File '%s' ignored because of a field with the same name", file.Name))
			}
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
