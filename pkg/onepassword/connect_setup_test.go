package onepassword

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var defaultNamespacedName = types.NamespacedName{Name: "onepassword-connect", Namespace: "default"}

func TestServiceSetup(t *testing.T) {

	// Register operator types with the runtime scheme.
	s := scheme.Scheme

	// Objects to track in the fake client.
	objs := []runtime.Object{}

	// Create a fake client to mock API calls.
	client := fake.NewFakeClientWithScheme(s, objs...)

	err := setupService(client, "../../deploy/connect/service.yaml")

	if err != nil {
		t.Errorf("Error Setting Up Connect: %v", err)
	}

	// check that service was created
	service := &corev1.Service{}
	err = client.Get(context.TODO(), defaultNamespacedName, service)
	if err != nil {
		t.Errorf("Error Setting Up Connect service: %v", err)
	}
}

func TestDeploymentSetup(t *testing.T) {

	// Register operator types with the runtime scheme.
	s := scheme.Scheme

	// Objects to track in the fake client.
	objs := []runtime.Object{}

	// Create a fake client to mock API calls.
	client := fake.NewFakeClientWithScheme(s, objs...)

	err := setupDeployment(client, "../../deploy/connect/deployment.yaml")

	if err != nil {
		t.Errorf("Error Setting Up Connect: %v", err)
	}

	// check that deployment was created
	deployment := &appsv1.Deployment{}
	err = client.Get(context.TODO(), defaultNamespacedName, deployment)
	if err != nil {
		t.Errorf("Error Setting Up Connect deployment: %v", err)
	}
}
