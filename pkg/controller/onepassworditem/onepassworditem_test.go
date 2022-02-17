package onepassworditem

import (
	"context"
	"fmt"
	"testing"

	"github.com/1Password/onepassword-operator/pkg/mocks"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"

	onepasswordv1 "github.com/1Password/onepassword-operator/pkg/apis/onepassword/v1"

	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	onePasswordItemKind       = "OnePasswordItem"
	onePasswordItemAPIVersion = "onepassword.com/v1"
	name                      = "test"
	namespace                 = "default"
	vaultId                   = "hfnjvi6aymbsnfc2xeeoheizda"
	itemId                    = "nwrhuano7bcwddcviubpp4mhfq"
	username                  = "test-user"
	password                  = "QmHumKc$mUeEem7caHtbaBaJ"
	firstHost                 = "http://localhost:8080"
	awsKey                    = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	iceCream                  = "freezing blue 20%"
	userKey                   = "username"
	passKey                   = "password"
	version                   = 123
)

type testReconcileItem struct {
	testName                string
	customResource          *onepasswordv1.OnePasswordItem
	existingSecret          *corev1.Secret
	expectedError           error
	expectedResultSecret    *corev1.Secret
	expectedEvents          []string
	opItem                  map[string]string
	existingOnePasswordItem *onepasswordv1.OnePasswordItem
}

var (
	expectedSecretData = map[string][]byte{
		"password": []byte(password),
		"username": []byte(username),
	}
	itemPath = fmt.Sprintf("vaults/%v/items/%v", vaultId, itemId)
)

var (
	time = metav1.Now()
)

var tests = []testReconcileItem{
	{
		testName: "Test Delete OnePasswordItem",
		customResource: &onepasswordv1.OnePasswordItem{
			TypeMeta: metav1.TypeMeta{
				Kind:       onePasswordItemKind,
				APIVersion: onePasswordItemAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Namespace:         namespace,
				DeletionTimestamp: &time,
				Finalizers: []string{
					finalizer,
				},
			},
			Spec: onepasswordv1.OnePasswordItemSpec{
				ItemPath: itemPath,
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation: fmt.Sprint(version),
				},
			},
			Data: expectedSecretData,
		},
		expectedError:        nil,
		expectedResultSecret: nil,
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
	},
	{
		testName: "Test Do not update if OnePassword Version has not changed",
		customResource: &onepasswordv1.OnePasswordItem{
			TypeMeta: metav1.TypeMeta{
				Kind:       onePasswordItemKind,
				APIVersion: onePasswordItemAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: onepasswordv1.OnePasswordItemSpec{
				ItemPath: itemPath,
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation:  fmt.Sprint(version),
					op.ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation:  fmt.Sprint(version),
					op.ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: "data we don't expect to have updated",
			passKey: "data we don't expect to have updated",
		},
	},
	{
		testName: "Test Updating Existing Kubernetes Secret using OnePasswordItem",
		customResource: &onepasswordv1.OnePasswordItem{
			TypeMeta: metav1.TypeMeta{
				Kind:       onePasswordItemKind,
				APIVersion: onePasswordItemAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation:  fmt.Sprint(version),
					op.ItemPathAnnotation: itemPath,
				},
				Labels: map[string]string{},
			},
			Spec: onepasswordv1.OnePasswordItemSpec{
				ItemPath: itemPath,
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation:  "456",
					op.ItemPathAnnotation: itemPath,
				},
				Labels: map[string]string{},
			},
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation:  fmt.Sprint(version),
					op.ItemPathAnnotation: itemPath,
				},
				Labels: map[string]string{},
			},
			Type: corev1.SecretTypeOpaque,
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
	},
	{
		testName: "Test Updating Type of Existing Kubernetes Secret using OnePasswordItem",
		customResource: &onepasswordv1.OnePasswordItem{
			TypeMeta: metav1.TypeMeta{
				Kind:       onePasswordItemKind,
				APIVersion: onePasswordItemAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation:  fmt.Sprint(version),
					op.ItemPathAnnotation: itemPath,
				},
				Labels: map[string]string{},
			},
			Spec: onepasswordv1.OnePasswordItemSpec{
				ItemPath: itemPath,
			},
			Type: string(corev1.SecretTypeBasicAuth),
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation:  fmt.Sprint(version),
					op.ItemPathAnnotation: itemPath,
				},
				Labels: map[string]string{},
			},
			Type: corev1.SecretTypeOpaque,
			Data: expectedSecretData,
		},
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation:  fmt.Sprint(version),
					op.ItemPathAnnotation: itemPath,
				},
				Labels: map[string]string{},
			},
			Type: corev1.SecretTypeBasicAuth,
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
	},
	{
		testName: "Custom secret type",
		customResource: &onepasswordv1.OnePasswordItem{
			TypeMeta: metav1.TypeMeta{
				Kind:       onePasswordItemKind,
				APIVersion: onePasswordItemAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: onepasswordv1.OnePasswordItemSpec{
				ItemPath: itemPath,
			},
			Type: "custom",
		},
		existingSecret: nil,
		expectedError:  nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation: fmt.Sprint(version),
				},
			},
			Type: corev1.SecretType("custom"),
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
	},
	{
		testName: "Secret from 1Password item with invalid K8s labels",
		customResource: &onepasswordv1.OnePasswordItem{
			TypeMeta: metav1.TypeMeta{
				Kind:       onePasswordItemKind,
				APIVersion: onePasswordItemAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "!my sECReT it3m%",
				Namespace: namespace,
			},
			Spec: onepasswordv1.OnePasswordItemSpec{
				ItemPath: itemPath,
			},
		},
		existingSecret: nil,
		expectedError:  nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret-it3m",
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation: fmt.Sprint(version),
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
	},
	{
		testName: "Secret from 1Password item with fields and sections that have invalid K8s labels",
		customResource: &onepasswordv1.OnePasswordItem{
			TypeMeta: metav1.TypeMeta{
				Kind:       onePasswordItemKind,
				APIVersion: onePasswordItemAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "!my sECReT it3m%",
				Namespace: namespace,
			},
			Spec: onepasswordv1.OnePasswordItemSpec{
				ItemPath: itemPath,
			},
		},
		existingSecret: nil,
		expectedError:  nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret-it3m",
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation: fmt.Sprint(version),
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"password":       []byte(password),
				"username":       []byte(username),
				"first-host":     []byte(firstHost),
				"AWS-Access-Key": []byte(awsKey),
				"ice-cream-type": []byte(iceCream),
			},
		},
		opItem: map[string]string{
			userKey:            username,
			passKey:            password,
			"first host":       firstHost,
			"AWS Access Key":   awsKey,
			"😄 ice-cream type": iceCream,
		},
	},
	{
		testName: "Secret from 1Password item with `-`, `_` and `.`",
		customResource: &onepasswordv1.OnePasswordItem{
			TypeMeta: metav1.TypeMeta{
				Kind:       onePasswordItemKind,
				APIVersion: onePasswordItemAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "!.my_sECReT.it3m%-_",
				Namespace: namespace,
			},
			Spec: onepasswordv1.OnePasswordItemSpec{
				ItemPath: itemPath,
			},
		},
		existingSecret: nil,
		expectedError:  nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-secret.it3m",
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation: fmt.Sprint(version),
				},
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"password":          []byte(password),
				"username":          []byte(username),
				"first-host":        []byte(firstHost),
				"AWS-Access-Key":    []byte(awsKey),
				"-_ice_cream.type.": []byte(iceCream),
			},
		},
		opItem: map[string]string{
			userKey:               username,
			passKey:               password,
			"first host":          firstHost,
			"AWS Access Key":      awsKey,
			"😄 -_ice_cream.type.": iceCream,
		},
	},
}

func TestReconcileOnePasswordItem(t *testing.T) {
	for _, testData := range tests {
		t.Run(testData.testName, func(t *testing.T) {

			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(onepasswordv1.SchemeGroupVersion, testData.customResource)

			// Objects to track in the fake client.
			objs := []runtime.Object{
				testData.customResource,
			}

			if testData.existingSecret != nil {
				objs = append(objs, testData.existingSecret)
			}

			if testData.existingOnePasswordItem != nil {
				objs = append(objs, testData.existingOnePasswordItem)
			}
			// Create a fake client to mock API calls.
			cl := fake.NewFakeClientWithScheme(s, objs...)
			// Create a OnePasswordItem object with the scheme and mock  kubernetes
			// and 1Password Connect client.

			opConnectClient := &mocks.TestClient{}
			mocks.GetGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {

				item := onepassword.Item{}
				item.Fields = []*onepassword.ItemField{}
				for k, v := range testData.opItem {
					item.Fields = append(item.Fields, &onepassword.ItemField{Label: k, Value: v})
				}
				item.Version = version
				item.Vault.ID = vaultUUID
				item.ID = uuid
				return &item, nil
			}
			r := &ReconcileOnePasswordItem{
				kubeClient:      cl,
				scheme:          s,
				opConnectClient: opConnectClient,
			}

			// Mock request to simulate Reconcile() being called on an event for a
			// watched resource .
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      testData.customResource.ObjectMeta.Name,
					Namespace: testData.customResource.ObjectMeta.Namespace,
				},
			}
			_, err := r.Reconcile(req)

			assert.Equal(t, testData.expectedError, err)

			var expectedSecretName string
			if testData.expectedResultSecret == nil {
				expectedSecretName = testData.customResource.Name
			} else {
				expectedSecretName = testData.expectedResultSecret.Name
			}

			// Check if Secret has been created and has the correct data
			secret := &corev1.Secret{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: expectedSecretName, Namespace: namespace}, secret)

			if testData.expectedResultSecret == nil {
				assert.Error(t, err)
				assert.True(t, errors2.IsNotFound(err))
			} else {
				assert.Equal(t, testData.expectedResultSecret.Data, secret.Data)
				assert.Equal(t, testData.expectedResultSecret.Name, secret.Name)
				assert.Equal(t, testData.expectedResultSecret.Type, secret.Type)
				assert.Equal(t, testData.expectedResultSecret.Annotations[op.VersionAnnotation], secret.Annotations[op.VersionAnnotation])

				updatedCR := &onepasswordv1.OnePasswordItem{}
				err = cl.Get(context.TODO(), req.NamespacedName, updatedCR)
				assert.NoError(t, err)
			}
		})
	}
}

func generateFields(username, password string) []*onepassword.ItemField {
	fields := []*onepassword.ItemField{
		{
			Label: "username",
			Value: username,
		},
		{
			Label: "password",
			Value: password,
		},
	}
	return fields
}
