package kubernetessecrets

import (
	"context"
	"fmt"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubeValidate "k8s.io/apimachinery/pkg/util/validation"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
)

const (
	restartDeploymentAnnotation = "false"
	testNamespace               = "test"
	testItemUUID                = "h46bb3jddvay7nxopfhvlwg35q"
	testVaultUUID               = "hfnjvi6aymbsnfc2xeeoheizda"
)

func TestCreateKubernetesSecretFromOnePasswordItem(t *testing.T) {
	ctx := context.Background()
	secretName := "test-secret-name"
	namespace := testNamespace

	item := model.Item{}
	item.Fields = generateFields(5)
	item.Version = 123
	item.VaultID = testVaultUUID
	item.ID = testItemUUID

	kubeClient := fake.NewClientBuilder().Build()
	secretLabels := map[string]string{}
	secretType := ""

	err := CreateKubernetesSecretFromItem(ctx, kubeClient, secretName, namespace, &item, restartDeploymentAnnotation,
		secretLabels, secretType, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	createdSecret := &corev1.Secret{}
	err = kubeClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, createdSecret)

	if err != nil {
		t.Errorf("Secret was not created: %v", err)
	}
	compareFields(item.Fields, createdSecret.Data, t)
	compareAnnotationsToItem(createdSecret.Annotations, item, t)
}

func TestKubernetesSecretFromOnePasswordItemOwnerReferences(t *testing.T) {
	ctx := context.Background()
	secretName := "test-secret-name"
	namespace := testNamespace

	item := model.Item{}
	item.Fields = generateFields(5)
	item.Version = 123
	item.VaultID = testVaultUUID
	item.ID = testItemUUID

	kubeClient := fake.NewClientBuilder().Build()
	secretLabels := map[string]string{}
	secretType := ""

	ownerRef := &metav1.OwnerReference{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		Name:       "test-deployment",
		UID:        types.UID("test-uid"),
	}
	err := CreateKubernetesSecretFromItem(ctx, kubeClient, secretName, namespace, &item, restartDeploymentAnnotation,
		secretLabels, secretType, ownerRef)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	createdSecret := &corev1.Secret{}
	err = kubeClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, createdSecret)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check owner references.
	gotOwnerRefs := createdSecret.ObjectMeta.OwnerReferences
	if len(gotOwnerRefs) != 1 {
		t.Errorf("Expected owner references length: 1 but got: %d", len(gotOwnerRefs))
	}

	expOwnerRef := metav1.OwnerReference{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		Name:       "test-deployment",
		UID:        types.UID("test-uid"),
	}
	gotOwnerRef := gotOwnerRefs[0]
	if gotOwnerRef != expOwnerRef {
		t.Errorf("Expected owner reference value: %v but got: %v", expOwnerRef, gotOwnerRef)
	}
}

func TestUpdateKubernetesSecretFromOnePasswordItem(t *testing.T) {
	ctx := context.Background()
	secretName := "test-secret-update"
	namespace := testNamespace

	item := model.Item{}
	item.Fields = generateFields(5)
	item.Version = 123
	item.VaultID = testVaultUUID
	item.ID = testItemUUID

	kubeClient := fake.NewClientBuilder().Build()
	secretLabels := map[string]string{}
	secretType := ""

	err := CreateKubernetesSecretFromItem(ctx, kubeClient, secretName, namespace, &item, restartDeploymentAnnotation,
		secretLabels, secretType, nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Updating kubernetes secret with new item
	newItem := model.Item{}
	newItem.Fields = generateFields(6)
	newItem.Version = 456
	newItem.VaultID = testVaultUUID
	newItem.ID = testItemUUID
	err = CreateKubernetesSecretFromItem(ctx, kubeClient, secretName, namespace, &newItem, restartDeploymentAnnotation,
		secretLabels, secretType, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	updatedSecret := &corev1.Secret{}
	err = kubeClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, updatedSecret)

	if err != nil {
		t.Errorf("Secret was not found: %v", err)
	}
	compareFields(newItem.Fields, updatedSecret.Data, t)
	compareAnnotationsToItem(updatedSecret.Annotations, newItem, t)
}
func TestBuildKubernetesSecretData(t *testing.T) {
	fields := generateFields(5)

	secretData := BuildKubernetesSecretData(fields, nil)
	if len(secretData) != len(fields) {
		t.Errorf("Unexpected number of secret fields returned. Expected 3, got %v", len(secretData))
	}
	compareFields(fields, secretData, t)
}

func TestBuildKubernetesSecretFromOnePasswordItem(t *testing.T) {
	annotationKey := "annotationKey"
	annotationValue := "annotationValue"
	name := "someName"
	namespace := "someNamespace"
	annotations := map[string]string{
		annotationKey: annotationValue,
	}
	item := model.Item{}
	item.Fields = generateFields(5)
	labels := map[string]string{}
	secretType := ""

	kubeSecret := BuildKubernetesSecretFromOnePasswordItem(name, namespace, annotations, labels, secretType, item, nil)
	if kubeSecret.Name != strings.ToLower(name) {
		t.Errorf("Expected name value: %v but got: %v", name, kubeSecret.Name)
	}
	if kubeSecret.Namespace != namespace {
		t.Errorf("Expected namespace value: %v but got: %v", namespace, kubeSecret.Namespace)
	}
	if kubeSecret.Annotations[annotationKey] != annotations[annotationKey] {
		t.Errorf("Expected namespace value: %v but got: %v", namespace, kubeSecret.Namespace)
	}
	compareFields(item.Fields, kubeSecret.Data, t)
}

func TestBuildKubernetesSecretFixesInvalidLabels(t *testing.T) {
	name := "inV@l1d k8s secret%name"
	expectedName := "inv-l1d-k8s-secret-name"
	namespace := "someNamespace"
	annotations := map[string]string{
		"annotationKey": "annotationValue",
	}
	labels := map[string]string{}
	item := model.Item{}
	secretType := ""

	item.Fields = []model.ItemField{
		{
			Label: "label w%th invalid ch!rs-",
			Value: "value1",
		},
		{
			Label: strings.Repeat("x", kubeValidate.DNS1123SubdomainMaxLength+1),
			Value: "name exceeds max length",
		},
	}

	kubeSecret := BuildKubernetesSecretFromOnePasswordItem(name, namespace, annotations, labels, secretType, item, nil)

	// Assert Secret's meta.name was fixed
	if kubeSecret.Name != expectedName {
		t.Errorf("Expected name value: %v but got: %v", name, kubeSecret.Name)
	}
	if kubeSecret.Namespace != namespace {
		t.Errorf("Expected namespace value: %v but got: %v", namespace, kubeSecret.Namespace)
	}

	// assert labels were fixed for each data key
	for key := range kubeSecret.Data {
		if !validLabel(key) {
			t.Errorf("Expected valid kubernetes label, got %s", key)
		}
	}
}

func TestCreateKubernetesTLSSecretFromOnePasswordItem(t *testing.T) {
	ctx := context.Background()
	secretName := "tls-test-secret-name"
	namespace := testNamespace

	item := model.Item{}
	item.Fields = generateFields(5)
	item.Version = 123
	item.VaultID = testVaultUUID
	item.ID = testItemUUID

	kubeClient := fake.NewClientBuilder().Build()
	secretLabels := map[string]string{}
	secretType := "kubernetes.io/tls"

	err := CreateKubernetesSecretFromItem(ctx, kubeClient, secretName, namespace, &item, restartDeploymentAnnotation,
		secretLabels, secretType, nil)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	createdSecret := &corev1.Secret{}
	err = kubeClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, createdSecret)

	if err != nil {
		t.Errorf("Secret was not created: %v", err)
	}

	if createdSecret.Type != corev1.SecretTypeTLS {
		t.Errorf("Expected secretType to be of tyype corev1.SecretTypeTLS, got %s", string(createdSecret.Type))
	}
}

func compareAnnotationsToItem(annotations map[string]string, item model.Item, t *testing.T) {
	actualVaultId, actualItemId, err := ParseVaultIdAndItemIdFromPath(annotations[ItemPathAnnotation])
	if err != nil {
		t.Errorf("Was unable to parse Item Path")
	}
	if actualVaultId != item.VaultID {
		t.Errorf("Expected annotation vault id to be %v but was %v", item.VaultID, actualVaultId)
	}
	if actualItemId != item.ID {
		t.Errorf("Expected annotation item id to be %v but was %v", item.ID, actualItemId)
	}
	if annotations[VersionAnnotation] != fmt.Sprint(item.Version) {
		t.Errorf("Expected annotation version to be %v but was %v", item.Version, annotations[VersionAnnotation])
	}

	if annotations[RestartDeploymentsAnnotation] != "false" {
		t.Errorf("Expected restart deployments annotation to be %v but was %v",
			restartDeploymentAnnotation, RestartDeploymentsAnnotation,
		)
	}
}

func compareFields(actualFields []model.ItemField, secretData map[string][]byte, t *testing.T) {
	for i := 0; i < len(actualFields); i++ {
		value, found := secretData[actualFields[i].Label]
		if !found {
			t.Errorf("Expected key %v is missing from secret data", actualFields[i].Label)
		}
		if string(value) != actualFields[i].Value {
			t.Errorf("Expected value %v but got %v", actualFields[i].Value, value)
		}
	}
}

func generateFields(numToGenerate int) []model.ItemField {
	fields := []model.ItemField{}
	for i := 0; i < numToGenerate; i++ {
		fields = append(fields, model.ItemField{
			Label: "key" + fmt.Sprint(i),
			Value: "value" + fmt.Sprint(i),
		})
	}
	return fields
}

func ParseVaultIdAndItemIdFromPath(path string) (string, string, error) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 4 && splitPath[0] == "vaults" && splitPath[2] == "items" {
		return splitPath[1], splitPath[3], nil
	}
	return "", "", fmt.Errorf(
		"%q is not an acceptable path for One Password item. Must be of the format: `vaults/{vault_id}/items/{item_id}`",
		path,
	)
}

func validLabel(v string) bool {
	if err := kubeValidate.IsConfigMapKey(v); len(err) > 0 {
		return false
	}
	return true
}
