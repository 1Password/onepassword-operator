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
	secretAnnotations := map[string]string{
		"testAnnotation": "exists",
	}
	err := CreateKubernetesSecretFromItem(ctx, kubeClient, secretName, namespace, &item, restartDeploymentAnnotation,
		secretLabels, secretAnnotations, secretType, nil)
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
	secretAnnotations := map[string]string{
		"testAnnotation": "exists",
	}

	ownerRef := &metav1.OwnerReference{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		Name:       "test-deployment",
		UID:        types.UID("test-uid"),
	}
	err := CreateKubernetesSecretFromItem(ctx, kubeClient, secretName, namespace, &item, restartDeploymentAnnotation,
		secretLabels, secretAnnotations, secretType, ownerRef)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	createdSecret := &corev1.Secret{}
	err = kubeClient.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, createdSecret)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check owner references.
	gotOwnerRefs := createdSecret.OwnerReferences
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
	secretAnnotations := map[string]string{
		"testAnnotation": "exists",
	}

	err := CreateKubernetesSecretFromItem(ctx, kubeClient, secretName, namespace, &item, restartDeploymentAnnotation,
		secretLabels, secretAnnotations, secretType, nil)

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
		secretLabels, secretAnnotations, secretType, nil)
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

	secretData := BuildKubernetesSecretData(fields, nil, nil)
	if len(secretData) != len(fields) {
		t.Errorf("Unexpected number of secret fields returned. Expected 3, got %v", len(secretData))
	}
	compareFields(fields, secretData, t)
}

func TestBuildKubernetesSecretDataWithEmptyValues(t *testing.T) {
	fields := []model.ItemField{
		{Label: "token", Value: "secret-token"},
		{Label: "runner-token", Value: ""},
		{Label: "another-field", Value: "value"},
		{Label: "empty-field-2", Value: ""},
	}

	secretData := BuildKubernetesSecretData(fields, nil, nil)

	// Verify all fields are present, including empty ones
	if len(secretData) != len(fields) {
		t.Errorf("Expected %d fields, got %d", len(fields), len(secretData))
	}

	for _, field := range fields {
		key := formatSecretDataName(field.Label)
		value, exists := secretData[key]
		if !exists {
			t.Errorf("Field '%s' should be present in secret data", field.Label)
			continue
		}
		if string(value) != field.Value {
			t.Errorf("Field '%s': expected value '%s', got '%s'", field.Label, field.Value, string(value))
		}
		// Verify empty values are empty byte slices (not nil)
		if field.Value == "" && len(value) != 0 {
			t.Errorf("Empty field '%s' should have empty byte slice, got length %d", field.Label, len(value))
		}
	}
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

func TestBuildKubernetesSecretDataWithURLs(t *testing.T) {
	fields := generateFields(2)
	urls := []model.ItemURL{
		{URL: "https://example.com", Label: "website", Primary: true},
		{URL: "https://support.example.com", Label: "support", Primary: false},
		{URL: "https://another.example.com", Label: "website", Primary: false},
	}

	secretData := BuildKubernetesSecretData(fields, urls, nil)

	// Should have fields + all URLs (both have different labels)
	if len(secretData) != 4 {
		t.Errorf("Expected 4 keys (2 fields + 2 URLs), got %d", len(secretData))
	}

	// Check primary URL is present and not the non-primary URL
	if string(secretData["website"]) != "https://example.com" {
		t.Errorf("Expected website URL, got %s", string(secretData["website"]))
	}

	// Check non-primary URL is also present (different label)
	if string(secretData["support"]) != "https://support.example.com" {
		t.Errorf("Expected support URL, got %s", string(secretData["support"]))
	}
}

func TestBuildKubernetesSecretDataWithFieldURLConflict(t *testing.T) {
	fields := []model.ItemField{
		{Label: "website", Value: "field-value-for-website"},
		{Label: "other-field", Value: "other-value"},
	}

	// Create a url with the same label "website" as field above - should be ignored
	urls := []model.ItemURL{
		{URL: "https://example.com", Label: "website", Primary: true},
		{URL: "https://support.example.com", Label: "support", Primary: false},
	}

	secretData := BuildKubernetesSecretData(fields, urls, nil)

	// Should have 2 fields + 1 url
	if len(secretData) != 3 {
		t.Errorf("Expected 3 keys (2 fields + 1 URL), got %d", len(secretData))
	}

	// Verify the field value is kept and not overwritten by url
	if string(secretData["website"]) != "field-value-for-website" {
		t.Errorf("Expected field value 'field-value-for-website', got %s", string(secretData["website"]))
	}

	if string(secretData["other-field"]) != "other-value" {
		t.Errorf("Expected 'other-value', got %s", string(secretData["other-field"]))
	}

	if string(secretData["support"]) != "https://support.example.com" {
		t.Errorf("Expected support URL, got %s", string(secretData["support"]))
	}
}

func TestBuildKubernetesSecretData_InvalidLabels(t *testing.T) {
	fields := []model.ItemField{
		{Label: "", Value: "empty-label"},
		{Label: "   ", Value: "whitespace-only"},
		{Label: "###", Value: "special-chars-only"},
		{Label: "@@@", Value: "at-signs-only"},
		{Label: "%%%", Value: "percent-signs-only"},
	}

	urls := []model.ItemURL{
		{URL: "https://example.com", Label: "", Primary: true},
		{URL: "https://test.com", Label: "   ", Primary: false},
		{URL: "https://other.com", Label: "###", Primary: false},
	}

	files := []model.File{
		{Name: ""},
		{Name: "   "},
		{Name: "###"},
	}
	files[0].SetContent([]byte("content1"))
	files[1].SetContent([]byte("content2"))
	files[2].SetContent([]byte("content3"))

	secretData := BuildKubernetesSecretData(fields, urls, files)

	if len(secretData) != 0 {
		t.Errorf("Expected 0 keys, got %d: %v", len(secretData), secretData)
	}
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
	secretAnnotations := map[string]string{
		"testAnnotation": "exists",
	}

	err := CreateKubernetesSecretFromItem(ctx, kubeClient, secretName, namespace, &item, restartDeploymentAnnotation,
		secretLabels, secretAnnotations, secretType, nil)
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
