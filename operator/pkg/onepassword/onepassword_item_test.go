package onepassword

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/1Password/onepassword-operator/operator/pkg/mocks"

	"github.com/1Password/connect-sdk-go/onepassword"
	onepasswordv1 "github.com/1Password/onepassword-operator/operator/pkg/apis/onepassword/v1"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type onepassworditemInjections struct {
	testName           string
	existingDeployment *appsv1.Deployment
	existingNamespace  *corev1.Namespace
	expectedError      error
	expectedEvents     []string
	opItem             map[string]string
	expectedOPItem     *onepasswordv1.OnePasswordItem
}

var onepassworditemTests = []onepassworditemInjections{
	{
		testName:          "Try to Create OnePasswordItem with container with valid op reference",
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
						Name:      name,
						Namespace: namespace,
						Annotations: map[string]string{
							ContainerInjectAnnotation: "test-app",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "test-app",
								Env: []corev1.EnvVar{
									{
										Name:  name,
										Value: fmt.Sprintf("op://%s/%s/test", vaultId, itemId),
									},
								},
							},
						},
					},
				},
			},
		},
		expectedError: nil,
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedOPItem: &onepasswordv1.OnePasswordItem{
			TypeMeta: metav1.TypeMeta{
				Kind:       "OnePasswordItem",
				APIVersion: "onepassword.com/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      injectedOnePasswordItemName,
				Namespace: namespace,
				Annotations: map[string]string{
					InjectedAnnotation: "true",
					VersionAnnotation:  "old",
				},
			},
			Spec: onepasswordv1.OnePasswordItemSpec{
				ItemPath: itemPath,
			},
		},
	},
	{
		testName:          "Container with no op:// reference does not create OnePasswordItem",
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
						Name:      name,
						Namespace: namespace,
						Annotations: map[string]string{
							ContainerInjectAnnotation: "test-app",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "test-app",
								Env: []corev1.EnvVar{
									{
										Name:  name,
										Value: fmt.Sprintf("some value"),
									},
								},
							},
						},
					},
				},
			},
		},
		expectedError: nil,
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedOPItem: nil,
	},
	{
		testName:          "Container with op:// reference missing vault and item does not create OnePasswordItem and returns error",
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
						Name:      name,
						Namespace: namespace,
						Annotations: map[string]string{
							ContainerInjectAnnotation: "test-app",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "test-app",
								Env: []corev1.EnvVar{
									{
										Name:  name,
										Value: fmt.Sprintf("op://"),
									},
								},
							},
						},
					},
				},
			},
		},
		expectedError: fmt.Errorf("Invalid secret reference : %s. Secret references should match op://<vault>/<item>/<field>", "op://"),
		opItem: map[string]string{
			userKey: username,
			passKey: password,
		},
		expectedOPItem: nil,
	},
}

func TestOnePasswordItemSecretInjected(t *testing.T) {
	for _, testData := range onepassworditemTests {
		t.Run(testData.testName, func(t *testing.T) {

			// Register operator types with the runtime scheme.
			s := scheme.Scheme
			s.AddKnownTypes(appsv1.SchemeGroupVersion, &onepasswordv1.OnePasswordItem{}, &onepasswordv1.OnePasswordItemList{}, &appsv1.Deployment{})

			// Objects to track in the fake client.
			objs := []runtime.Object{
				testData.existingDeployment,
				testData.existingNamespace,
			}

			// Create a fake client to mock API calls.
			cl := fake.NewFakeClientWithScheme(s, objs...)

			opConnectClient := &mocks.TestClient{}
			mocks.GetGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {

				item := onepassword.Item{}
				item.Fields = generateFields(testData.opItem["username"], testData.opItem["password"])
				item.Version = itemVersion
				item.Vault.ID = vaultUUID
				item.ID = uuid
				return &item, nil
			}

			injectedContainers := testData.existingDeployment.Spec.Template.ObjectMeta.Annotations[ContainerInjectAnnotation]
			parsedInjectedContainers := strings.Split(injectedContainers, ",")
			err := CreateOnePasswordItemResourceFromDeployment(opConnectClient, cl, testData.existingDeployment, parsedInjectedContainers)

			assert.Equal(t, testData.expectedError, err)

			// Check if Secret has been created and has the correct data
			opItemCR := &onepasswordv1.OnePasswordItem{}
			err = cl.Get(context.TODO(), types.NamespacedName{Name: injectedOnePasswordItemName, Namespace: namespace}, opItemCR)

			if testData.expectedOPItem == nil {
				assert.Error(t, err)
				assert.True(t, errors2.IsNotFound(err))
			} else {
				assert.Equal(t, testData.expectedOPItem.Spec.ItemPath, opItemCR.Spec.ItemPath)
				assert.Equal(t, testData.expectedOPItem.Name, opItemCR.Name)
				assert.Equal(t, testData.expectedOPItem.Annotations[InjectedAnnotation], opItemCR.Annotations[InjectedAnnotation])
			}

		})
	}
}
