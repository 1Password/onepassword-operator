package kubernetessecrets

import (
	"context"
	"fmt"
	"strings"
	"testing"

	kubeValidate "k8s.io/apimachinery/pkg/util/validation"

	"github.com/1Password/connect-sdk-go/onepassword"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const restartDeploymentAnnotation = "false"

type k8s struct {
	clientset kubernetes.Interface
}

func TestCreateKubernetesSecretFromOnePasswordItem(t *testing.T) {
	secretName := "test-secret-name"
	namespace := "test"

	item := onepassword.Item{}
	item.Fields = generateFields(5)
	item.Version = 123
	item.Vault.ID = "hfnjvi6aymbsnfc2xeeoheizda"
	item.ID = "h46bb3jddvay7nxopfhvlwg35q"

	kubeClient := fake.NewFakeClient()
	err := CreateKubernetesSecretFromItem(kubeClient, secretName, namespace, &item, restartDeploymentAnnotation)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	createdSecret := &corev1.Secret{}
	err = kubeClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: namespace}, createdSecret)

	if err != nil {
		t.Errorf("Secret was not created: %v", err)
	}
	compareFields(item.Fields, createdSecret.Data, t)
	compareAnnotationsToItem(item.Vault.ID, item.ID, createdSecret.Annotations, item, t)
}

func TestUpdateKubernetesSecretFromOnePasswordItem(t *testing.T) {
	secretName := "test-secret-update"
	namespace := "test"

	item := onepassword.Item{}
	item.Fields = generateFields(5)
	item.Version = 123
	item.Vault.ID = "hfnjvi6aymbsnfc2xeeoheizda"
	item.ID = "h46bb3jddvay7nxopfhvlwg35q"

	kubeClient := fake.NewFakeClient()
	err := CreateKubernetesSecretFromItem(kubeClient, secretName, namespace, &item, restartDeploymentAnnotation)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Updating kubernetes secret with new item
	newItem := onepassword.Item{}
	newItem.Fields = generateFields(6)
	newItem.Version = 456
	newItem.Vault.ID = "hfnjvi6aymbsnfc2xeeoheizda"
	newItem.ID = "h46bb3jddvay7nxopfhvlwg35q"
	err = CreateKubernetesSecretFromItem(kubeClient, secretName, namespace, &newItem, restartDeploymentAnnotation)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	updatedSecret := &corev1.Secret{}
	err = kubeClient.Get(context.Background(), types.NamespacedName{Name: secretName, Namespace: namespace}, updatedSecret)

	if err != nil {
		t.Errorf("Secret was not found: %v", err)
	}
	compareFields(newItem.Fields, updatedSecret.Data, t)
	compareAnnotationsToItem(newItem.Vault.ID, newItem.ID, updatedSecret.Annotations, newItem, t)
}
func TestBuildKubernetesSecretData(t *testing.T) {
	fields := generateFields(5)

	secretData := BuildKubernetesSecretData(fields)
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
	item := onepassword.Item{}
	item.Fields = generateFields(5)

	kubeSecret := BuildKubernetesSecretFromOnePasswordItem(name, namespace, annotations, item)
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
	item := onepassword.Item{}

	item.Fields = []*onepassword.ItemField{
		{
			Label: "label w%th invalid ch!rs-",
			Value: "value1",
		},
		{
			Label: strings.Repeat("x", kubeValidate.DNS1123SubdomainMaxLength+1),
			Value: "name exceeds max length",
		},
	}

	kubeSecret := BuildKubernetesSecretFromOnePasswordItem(name, namespace, annotations, item)

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

func compareAnnotationsToItem(actualVaultId, actualItemId string, annotations map[string]string, item onepassword.Item, t *testing.T) {
	if actualVaultId != item.Vault.ID {
		t.Errorf("Expected annotation vault id to be %v but was %v", item.Vault.ID, actualVaultId)
	}
	if actualItemId != item.ID {
		t.Errorf("Expected annotation item id to be %v but was %v", item.ID, actualItemId)
	}
	if annotations[VersionAnnotation] != fmt.Sprint(item.Version) {
		t.Errorf("Expected annotation version to be %v but was %v", item.Version, annotations[VersionAnnotation])
	}

	if annotations[RestartDeploymentsAnnotation] != "false" {
		t.Errorf("Expected restart deployments annotation to be %v but was %v", restartDeploymentAnnotation, RestartDeploymentsAnnotation)
	}
}

func compareFields(actualFields []*onepassword.ItemField, secretData map[string][]byte, t *testing.T) {
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

func generateFields(numToGenerate int) []*onepassword.ItemField {
	fields := []*onepassword.ItemField{}
	for i := 0; i < numToGenerate; i++ {
		field := onepassword.ItemField{
			Label: "key" + fmt.Sprint(i),
			Value: "value" + fmt.Sprint(i),
		}
		fields = append(fields, &field)
	}
	return fields
}

func validLabel(v string) bool {
	if err := kubeValidate.IsDNS1123Subdomain(v); len(err) > 0 {
		return false
	}
	return true
}
