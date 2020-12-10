package deployment

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/1Password/onepassword-operator/pkg/mocks"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"

	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
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
	deploymentKind       = "Deployment"
	deploymentAPIVersion = "v1"
	name                 = "test-deployment"
	namespace            = "default"
	vaultId              = "hfnjvi6aymbsnfc2xeeoheizda"
	itemId               = "nwrhuano7bcwddcviubpp4mhfq"
	username             = "test-user"
	password             = "QmHumKc$mUeEem7caHtbaBaJ"
	userKey              = "username"
	passKey              = "password"
	version              = 123
)

type testReconcileItem struct {
	testName             string
	deploymentResource   *appsv1.Deployment
	existingSecret       *corev1.Secret
	expectedError        error
	expectedResultSecret *corev1.Secret
	expectedEvents       []string
	opItem               map[string]string
	existingDeployment   *appsv1.Deployment
}

var (
	expectedSecretData = map[string][]byte{
		"password": []byte(password),
		"username": []byte(username),
	}
	itemPath = fmt.Sprintf("vaults/%v/items/%v", vaultId, itemId)
)

var (
	time     = metav1.Now()
	regex, _ = regexp.Compile(annotationRegExpString)
)

var tests = []testReconcileItem{
	{
		testName: "Test Delete Deployment where secret is being used in another deployment's volumes",
		deploymentResource: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Namespace:         namespace,
				DeletionTimestamp: &time,
				Finalizers: []string{
					finalizer,
				},
				Annotations: map[string]string{
					op.ItemPathAnnotation: itemPath,
					op.NameAnnotation:     name,
				},
			},
		},
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "another-deployment",
				Namespace: namespace,
				Annotations: map[string]string{
					op.ItemPathAnnotation: itemPath,
					op.NameAnnotation:     name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Volumes: []corev1.Volume{
							{
								Name: name,
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: name,
									},
								},
							},
						},
					},
				},
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
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation: fmt.Sprint(version),
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
	},
	{
		testName: "Test Delete Deployment where secret is being used in another deployment's container",
		deploymentResource: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Namespace:         namespace,
				DeletionTimestamp: &time,
				Finalizers: []string{
					finalizer,
				},
				Annotations: map[string]string{
					op.ItemPathAnnotation: itemPath,
					op.NameAnnotation:     name,
				},
			},
		},
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "another-deployment",
				Namespace: namespace,
				Annotations: map[string]string{
					op.ItemPathAnnotation: itemPath,
					op.NameAnnotation:     name,
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Env: []corev1.EnvVar{
									{
										Name: name,
										ValueFrom: &corev1.EnvVarSource{
											SecretKeyRef: &corev1.SecretKeySelector{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: name,
												},
												Key: passKey,
											},
										},
									},
								},
							},
						},
					},
				},
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
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation: fmt.Sprint(version),
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
	},
	{
		testName: "Test Delete Deployment",
		deploymentResource: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              name,
				Namespace:         namespace,
				DeletionTimestamp: &time,
				Finalizers: []string{
					finalizer,
				},
				Annotations: map[string]string{
					op.ItemPathAnnotation: itemPath,
					op.NameAnnotation:     name,
				},
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
		testName: "Test Do not update if OnePassword Item Version has not changed",
		deploymentResource: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.ItemPathAnnotation: itemPath,
					op.NameAnnotation:     name,
				},
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
		expectedError: nil,
		expectedResultSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation: fmt.Sprint(version),
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
		testName: "Test Updating Existing Kubernetes Secret using Deployment",
		deploymentResource: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.ItemPathAnnotation: itemPath,
					op.NameAnnotation:     name,
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.VersionAnnotation: "456",
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
					op.VersionAnnotation: fmt.Sprint(version),
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
	},
	{
		testName: "Create Deployment",
		deploymentResource: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					op.ItemPathAnnotation: itemPath,
					op.NameAnnotation:     name,
				},
			},
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
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
	},
}

func TestReconcileDepoyment(t *testing.T) {
	for _, testData := range tests {
		t.Run(testData.testName, func(t *testing.T) {

			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(appsv1.SchemeGroupVersion, testData.deploymentResource)

			// Objects to track in the fake client.
			objs := []runtime.Object{
				testData.deploymentResource,
			}

			if testData.existingSecret != nil {
				objs = append(objs, testData.existingSecret)
			}

			if testData.existingDeployment != nil {
				objs = append(objs, testData.existingDeployment)
			}

			// Create a fake client to mock API calls.
			cl := fake.NewFakeClientWithScheme(s, objs...)
			// Create a Deployment object with the scheme and mock  kubernetes
			// and 1Password Connect client.

			opConnectClient := &mocks.TestClient{}
			mocks.GetGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {

				item := onepassword.Item{}
				item.Fields = generateFields(testData.opItem["username"], testData.opItem["password"])
				item.Version = version
				item.Vault.ID = vaultUUID
				item.ID = uuid
				return &item, nil
			}
			r := &ReconcileDeployment{
				kubeClient:         cl,
				scheme:             s,
				opConnectClient:    opConnectClient,
				opAnnotationRegExp: regex,
			}

			// Mock request to simulate Reconcile() being called on an event for a
			// watched resource .
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      name,
					Namespace: namespace,
				},
			}
			_, err := r.Reconcile(req)

			assert.Equal(t, testData.expectedError, err)

			var expectedSecretName string
			if testData.expectedResultSecret == nil {
				expectedSecretName = testData.deploymentResource.Name
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

				updatedCR := &appsv1.Deployment{}
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
