package onepassword

import (
	"context"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logConnectSetup = logf.Log.WithName("ConnectSetup")
var deploymentPath = "../config/connect/deployment.yaml"
var servicePath = "../config/connect/service.yaml"

func SetupConnect(kubeClient client.Client, deploymentNamespace string) error {
	err := setupService(kubeClient, servicePath, deploymentNamespace)
	if err != nil {
		return err
	}

	err = setupDeployment(kubeClient, deploymentPath, deploymentNamespace)
	if err != nil {
		return err
	}

	return nil
}

func setupDeployment(kubeClient client.Client, deploymentPath string, deploymentNamespace string) error {
	existingDeployment := &appsv1.Deployment{}

	// check if deployment has already been created
	err := kubeClient.Get(context.Background(), types.NamespacedName{Name: "onepassword-connect", Namespace: deploymentNamespace}, existingDeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			logConnectSetup.Info("No existing Connect deployment found. Creating Deployment")
			return createDeployment(kubeClient, deploymentPath, deploymentNamespace)
		}
	}
	return err
}

func createDeployment(kubeClient client.Client, deploymentPath string, deploymentNamespace string) error {
	deployment, err := getDeploymentToCreate(deploymentPath, deploymentNamespace)
	if err != nil {
		return err
	}

	err = kubeClient.Create(context.Background(), deployment)
	if err != nil {
		return err
	}

	return nil
}

func getDeploymentToCreate(deploymentPath string, deploymentNamespace string) (*appsv1.Deployment, error) {
	f, err := os.Open(deploymentPath)
	if err != nil {
		return nil, err
	}
	deployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Namespace: deploymentNamespace,
		},
	}

	err = yaml.NewYAMLOrJSONDecoder(f, 4096).Decode(deployment)
	if err != nil {
		return nil, err
	}
	return deployment, nil
}

func setupService(kubeClient client.Client, servicePath string, deploymentNamespace string) error {
	existingService := &corev1.Service{}

	//check if service has already been created
	err := kubeClient.Get(context.Background(), types.NamespacedName{Name: "onepassword-connect", Namespace: deploymentNamespace}, existingService)
	if err != nil {
		if errors.IsNotFound(err) {
			logConnectSetup.Info("No existing Connect service found. Creating Service")
			return createService(kubeClient, servicePath, deploymentNamespace)
		}
	}
	return err
}

func createService(kubeClient client.Client, servicePath string, deploymentNamespace string) error {
	f, err := os.Open(servicePath)
	if err != nil {
		return err
	}
	service := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Namespace: deploymentNamespace,
		},
	}

	err = yaml.NewYAMLOrJSONDecoder(f, 4096).Decode(service)
	if err != nil {
		return err
	}

	err = kubeClient.Create(context.Background(), service)
	if err != nil {
		return err
	}

	return nil
}
