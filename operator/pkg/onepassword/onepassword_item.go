package onepassword

import (
	"context"
	"fmt"

	"github.com/1Password/connect-sdk-go/connect"
	onepasswordv1 "github.com/1Password/onepassword-operator/operator/pkg/apis/onepassword/v1"
	"github.com/1Password/onepassword-operator/operator/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	kubernetesClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateOnePasswordItemResourceFromDeployment(opClient connect.Client, kubeClient kubernetesClient.Client, deployment *appsv1.Deployment, injectedContainers []string) error {
	containers := deployment.Spec.Template.Spec.Containers
	containers = append(containers, deployment.Spec.Template.Spec.InitContainers...)
	for _, container := range containers {
		// check if container is listed is one of the containers
		// set to have injected secrets
		for _, injectedContainer := range injectedContainers {
			if injectedContainer != container.Name {
				continue
			}
			// create a one password item custom resource to track updates for injected secrets
			err := CreateOnePasswordCRSecretsFromContainer(opClient, kubeClient, container, deployment.Namespace)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func CreateOnePasswordCRSecretsFromContainer(opClient connect.Client, kubeClient kubernetesClient.Client, container corev1.Container, namespace string) error {
	for _, env := range container.Env {
		// if value is not of format op://<vault>/<item>/<field> then ignore
		vault, item, err := ParseReference(env.Value)
		if err != nil {
			continue
		}
		// create a one password item custom resource to track updates for injected secrets
		err = CreateOnePasswordCRSecretFromReference(opClient, kubeClient, vault, item, namespace)
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateOnePasswordCRSecretFromReference(opClient connect.Client, kubeClient kubernetesClient.Client, vault, item, namespace string) error {

	retrievedItem, err := GetOnePasswordItemByPath(opClient, fmt.Sprintf("vaults/%s/items/%s", vault, item))
	if err != nil {
		return fmt.Errorf("Failed to retrieve item: %v", err)
	}

	name := utils.BuildInjectedOnePasswordItemName(vault, item)
	onepassworditem := BuildOnePasswordItemCRFromPath(vault, item, name, namespace, fmt.Sprint(retrievedItem.Version))

	currentOnepassworditem := &onepasswordv1.OnePasswordItem{}
	err = kubeClient.Get(context.Background(), types.NamespacedName{Name: onepassworditem.Name, Namespace: onepassworditem.Namespace}, currentOnepassworditem)
	if errors.IsNotFound(err) {
		log.Info(fmt.Sprintf("Creating OnePasswordItem CR %v at namespace '%v'", onepassworditem.Name, onepassworditem.Namespace))
		return kubeClient.Create(context.Background(), onepassworditem)
	} else if err != nil {
		return err
	}
	return nil
}

func BuildOnePasswordItemCRFromPath(vault, item, name, namespace, version string) *onepasswordv1.OnePasswordItem {
	return &onepasswordv1.OnePasswordItem{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				InjectedAnnotation: "true",
				VersionAnnotation:  version,
			},
		},
		Spec: onepasswordv1.OnePasswordItemSpec{
			ItemPath: fmt.Sprintf("vaults/%s/items/%s", vault, item),
		},
	}
}
