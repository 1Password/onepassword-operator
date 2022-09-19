package controllers

import (
	"context"

	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/1Password/onepassword-operator/pkg/mocks"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	onepasswordv1 "github.com/1Password/onepassword-operator/api/v1"
)

var _ = Describe("Deployment controller", func() {
	const (
		deploymentKind       = "Deployment"
		deploymentAPIVersion = "v1"
		deploymentName       = "test-deployment"
	)

	BeforeEach(func() {
		// failed test runs that don't clean up leave resources behind.
		k8sClient.DeleteAllOf(context.Background(), &onepasswordv1.OnePasswordItem{}, client.InNamespace(namespace))
		k8sClient.DeleteAllOf(context.Background(), &v1.Secret{}, client.InNamespace(namespace))
		k8sClient.DeleteAllOf(context.Background(), &appsv1.Deployment{}, client.InNamespace(namespace))

		mocks.DoGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {

			item := onepassword.Item{}
			item.Fields = []*onepassword.ItemField{}
			for k, v := range itemData {
				item.Fields = append(item.Fields, &onepassword.ItemField{Label: k, Value: v})
			}
			item.Version = version
			item.Vault.ID = vaultUUID
			item.ID = uuid
			return &item, nil
		}
	})

	// TODO: Implement the following test cases:
	//  - Updating Existing K8s Secret using Deployment
	//  - Do not update if Annotations have not changed
	//  - Delete Deployment where secret is being used in another deployment's container
	//  - Delete Deployment where secret is being used in another deployment's volumes

	Context("Deployment with secrets from 1Password", func() {
		It("Should Handle a deployment correctly", func() {
			ctx := context.Background()

			deploymentKey := types.NamespacedName{
				Name:      deploymentName,
				Namespace: namespace,
			}

			secretKey := types.NamespacedName{
				Name:      ItemName,
				Namespace: namespace,
			}

			By("Deploying a pod with proper annotations successfully")
			deploymentResource := &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind:       deploymentKind,
					APIVersion: deploymentAPIVersion,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentKey.Name,
					Namespace: deploymentKey.Namespace,
					Annotations: map[string]string{
						op.ItemPathAnnotation: itemPath,
						op.NameAnnotation:     ItemName,
					},
				},
				Spec: appsv1.DeploymentSpec{
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": deploymentName},
						},
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:            deploymentName,
									Image:           "eu.gcr.io/kyma-project/example/http-db-service:0.0.6",
									ImagePullPolicy: "IfNotPresent",
								},
							},
						},
					},
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": deploymentName},
					},
				},
			}
			Expect(k8sClient.Create(ctx, deploymentResource)).Should(Succeed())

			By("Creating the K8s secret successfully")
			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretKey, createdSecret)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdSecret.Data).Should(Equal(expectedSecretData))

			By("Deleting the pod")
			Eventually(func() error {
				f := &appsv1.Deployment{}
				err := k8sClient.Get(ctx, deploymentKey, f)
				if err != nil {
					return err
				}
				return k8sClient.Delete(ctx, f)
			}, timeout, interval).Should(Succeed())

			Eventually(func() error {
				f := &appsv1.Deployment{}
				return k8sClient.Get(ctx, deploymentKey, f)
			}, timeout, interval).ShouldNot(Succeed())

			Eventually(func() error {
				f := &v1.Secret{}
				return k8sClient.Get(ctx, secretKey, f)
			}, timeout, interval).ShouldNot(Succeed())
		})
	})
})