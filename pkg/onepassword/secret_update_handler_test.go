package onepassword

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/1Password/onepassword-operator/pkg/mocks"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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
	itemVersion          = 123
)

type testUpdateSecretTask struct {
	testName                 string
	existingDeployment       *appsv1.Deployment
	existingNamespace        *corev1.Namespace
	existingSecret           *corev1.Secret
	expectedError            error
	expectedResultSecret     *corev1.Secret
	opItem                   map[string]string
	expectedRestart          bool
	globalAutoRestartEnabled bool
}

var (
	expectedSecretData = map[string][]byte{
		"password": []byte(password),
		"username": []byte(username),
	}
	itemPath = fmt.Sprintf("vaults/%v/items/%v", vaultId, itemId)
)

var defaultNamespace = &corev1.Namespace{
	ObjectMeta: metav1.ObjectMeta{
		Name: namespace,
	},
}

// TODO: Refactor test cases to avoid duplication.
//
//nolint:dupl
var tests = []testUpdateSecretTask{
	{
		testName:          "Test unrelated deployment is not restarted with an updated secret",
		existingNamespace: defaultNamespace,
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					NameAnnotation:     "unlrelated secret",
					ItemPathAnnotation: itemPath,
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
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
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          false,
		globalAutoRestartEnabled: true,
	},
	{
		testName:          "OP item has new version. Secret needs update. Deployment is restarted based on containers",
		existingNamespace: defaultNamespace,
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
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
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
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
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: true,
	},
	{
		testName:          "OP item has new version. Secret needs update. Deployment is restarted based on annotation",
		existingNamespace: defaultNamespace,
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					ItemPathAnnotation: itemPath,
					NameAnnotation:     name,
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
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
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: true,
	},
	{
		testName:          "OP item has new version. Secret needs update. Deployment is restarted based on volume",
		existingNamespace: defaultNamespace,
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
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
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
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
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: true,
	},
	{
		testName:          "No secrets need update. No deployment is restarted",
		existingNamespace: defaultNamespace,
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					ItemPathAnnotation: itemPath,
					NameAnnotation:     name,
				},
			},
		},
		existingSecret: &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
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
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          false,
		globalAutoRestartEnabled: true,
	},
	{
		testName: `Deployment is not restarted when no auto restart is set to true for all
		deployments and is not overwritten by by a namespace or deployment annotation`,
		existingNamespace: defaultNamespace,
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
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
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
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
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          false,
		globalAutoRestartEnabled: false,
	},
	{
		testName:          `Secret autostart true value takes precedence over false deployment value`,
		existingNamespace: defaultNamespace,
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					RestartDeploymentsAnnotation: "false",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
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
					VersionAnnotation:            "old version",
					ItemPathAnnotation:           itemPath,
					RestartDeploymentsAnnotation: "true",
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
					VersionAnnotation:            fmt.Sprint(itemVersion),
					ItemPathAnnotation:           itemPath,
					RestartDeploymentsAnnotation: "true",
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: false,
	},
	{
		testName:          `Secret autostart true value takes precedence over false deployment value`,
		existingNamespace: defaultNamespace,
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					RestartDeploymentsAnnotation: "true",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
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
					VersionAnnotation:            "old version",
					ItemPathAnnotation:           itemPath,
					RestartDeploymentsAnnotation: "false",
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
					VersionAnnotation:            fmt.Sprint(itemVersion),
					ItemPathAnnotation:           itemPath,
					RestartDeploymentsAnnotation: "false",
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          false,
		globalAutoRestartEnabled: true,
	},
	{
		testName:          `Deployment autostart true value takes precedence over false global auto restart value`,
		existingNamespace: defaultNamespace,
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					RestartDeploymentsAnnotation: "true",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
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
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
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
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: false,
	},
	{
		testName: `Deployment autostart false value takes precedence over false global auto restart value,
		 and true namespace value.`,
		existingNamespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Annotations: map[string]string{
					RestartDeploymentsAnnotation: "true",
				},
			},
		},
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Annotations: map[string]string{
					RestartDeploymentsAnnotation: "false",
				},
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
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
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
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
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          false,
		globalAutoRestartEnabled: false,
	},
	{
		testName: `Namespace autostart true value takes precedence over false global auto restart value`,
		existingNamespace: &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
				Annotations: map[string]string{
					RestartDeploymentsAnnotation: "true",
				},
			},
		},
		existingDeployment: &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       deploymentKind,
				APIVersion: deploymentAPIVersion,
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{"external-annotation": "some-value"},
					},
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
					VersionAnnotation:  "old version",
					ItemPathAnnotation: itemPath,
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
					VersionAnnotation:  fmt.Sprint(itemVersion),
					ItemPathAnnotation: itemPath,
				},
			},
			Data: expectedSecretData,
		},
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedRestart:          true,
		globalAutoRestartEnabled: false,
	},
}

func TestUpdateSecretHandler(t *testing.T) {
	for _, testData := range tests {
		t.Run(testData.testName, func(t *testing.T) {
			ctx := context.Background()
			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(appsv1.SchemeGroupVersion, testData.existingDeployment)

			// Objects to track in the fake client.
			objs := []runtime.Object{
				testData.existingDeployment,
				testData.existingNamespace,
			}

			if testData.existingSecret != nil {
				objs = append(objs, testData.existingSecret)
			}

			// Create a fake client to mock API calls.
			cl := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

			mockOpClient := &mocks.TestClient{}
			mockOpClient.On("GetItemByID", mock.Anything, mock.Anything).Return(createItem(), nil)
			h := &SecretUpdateHandler{
				client:                             cl,
				opClient:                           mockOpClient,
				shouldAutoRestartDeploymentsGlobal: testData.globalAutoRestartEnabled,
			}

			err := h.UpdateKubernetesSecretsTask(ctx)

			assert.Equal(t, testData.expectedError, err)

			var expectedSecretName string
			if testData.expectedResultSecret == nil {
				expectedSecretName = testData.existingDeployment.Name
			} else {
				expectedSecretName = testData.expectedResultSecret.Name
			}

			// Check if Secret has been created and has the correct data
			secret := &corev1.Secret{}
			err = cl.Get(ctx, types.NamespacedName{Name: expectedSecretName, Namespace: namespace}, secret)

			if testData.expectedResultSecret == nil {
				assert.Error(t, err)
				assert.True(t, errors2.IsNotFound(err))
			} else {
				assert.Equal(t, testData.expectedResultSecret.Data, secret.Data)
				assert.Equal(t, testData.expectedResultSecret.Name, secret.Name)
				assert.Equal(t, testData.expectedResultSecret.Type, secret.Type)
				assert.Equal(t, testData.expectedResultSecret.Annotations[VersionAnnotation], secret.Annotations[VersionAnnotation])
			}

			// check if deployment has been restarted
			deployment := &appsv1.Deployment{}
			err = cl.Get(ctx, types.NamespacedName{Name: testData.existingDeployment.Name, Namespace: namespace}, deployment)
			assert.NoError(t, err)

			_, ok := deployment.Spec.Template.Annotations[RestartAnnotation]
			if ok {
				assert.True(t, testData.expectedRestart, "Expected deployment to restart but it did not")
			} else {
				assert.False(t, testData.expectedRestart, "Deployment was restarted but should not have been.")
			}

			oldPodTemplateAnnotations := testData.existingDeployment.Spec.Template.Annotations
			newPodTemplateAnnotations := deployment.Spec.Template.Annotations
			for name, expected := range oldPodTemplateAnnotations {
				actual, ok := newPodTemplateAnnotations[name]
				if assert.Truef(t, ok, "Annotation %s was present in original pod template but was dropped after update", name) {
					assert.Equalf(t, expected, actual, "Annotation value for %s original pod template has changed", name)
					continue
				}
			}
		})
	}
}

func TestIsUpdatedSecret(t *testing.T) {

	secretName := "test-secret"
	updatedSecrets := map[string]*corev1.Secret{
		"some_secret": {},
	}
	assert.False(t, isUpdatedSecret(secretName, updatedSecrets))

	updatedSecrets[secretName] = &corev1.Secret{}
	assert.True(t, isUpdatedSecret(secretName, updatedSecrets))
}

func createItem() *model.Item {
	return &model.Item{
		ID:      itemId,
		VaultID: vaultId,
		Version: itemVersion,
		Tags:    []string{"tag1", "tag2"},
		Fields: []model.ItemField{
			{
				Label: "username",
				Value: username,
			},
			{
				Label: "password",
				Value: password,
			},
		},
		Files:     []model.File{},
		CreatedAt: time.Now(),
	}
}
