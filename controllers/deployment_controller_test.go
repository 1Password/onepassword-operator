package controllers

import (
	"context"
	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/1Password/onepassword-operator/pkg/mocks"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	onepasswordv1 "github.com/1Password/onepassword-operator/api/v1"
)

const (
	deploymentKind       = "Deployment"
	deploymentAPIVersion = "v1"
	deploymentName       = "test-deployment"
)

var _ = Describe("Deployment controller", func() {
	var ctx context.Context
	var deploymentKey types.NamespacedName
	var secretKey types.NamespacedName
	var deploymentResource *appsv1.Deployment
	createdSecret := &v1.Secret{}

	MakeDeployment := func() {
		ctx = context.Background()

		deploymentKey = types.NamespacedName{
			Name:      deploymentName,
			Namespace: namespace,
		}

		secretKey = types.NamespacedName{
			Name:      ItemName,
			Namespace: namespace,
		}

		By("Deploying a pod with proper annotations successfully")
		deploymentResource = &appsv1.Deployment{
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
		time.Sleep(time.Millisecond * 100)
		Eventually(func() bool {
			err := k8sClient.Get(ctx, secretKey, createdSecret)
			if err != nil {
				return false
			}
			return true
		}, timeout, interval).Should(BeTrue())
		Expect(createdSecret.Data).Should(Equal(expectedSecretData))
	}

	var CleanK8sResources = func() {
		// failed test runs that don't clean up leave resources behind.
		err := k8sClient.DeleteAllOf(context.Background(), &onepasswordv1.OnePasswordItem{}, client.InNamespace(namespace))
		Expect(err).ToNot(HaveOccurred())

		err2 := k8sClient.DeleteAllOf(context.Background(), &v1.Secret{}, client.InNamespace(namespace))
		Expect(err2).ToNot(HaveOccurred())

		err3 := k8sClient.DeleteAllOf(context.Background(), &appsv1.Deployment{}, client.InNamespace(namespace))
		Expect(err3).ToNot(HaveOccurred())
	}

	var MockGetItemFunc = func() {
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
	}

	BeforeEach(func() {
		CleanK8sResources()
		MockGetItemFunc()
		time.Sleep(time.Second) // TODO: can we achieve that with ginkgo?
		MakeDeployment()
	})

	Context("Deployment with secrets from 1Password", func() {
		It("Should Handle a deployment correctly", func() {
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

		It("Should update existing K8s Secret using deployment", func() {
			By("Updating secret")
			mocks.DoGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {
				item := onepassword.Item{}
				item.Fields = []*onepassword.ItemField{}
				for k, v := range itemData2 {
					item.Fields = append(item.Fields, &onepassword.ItemField{Label: k, Value: v})
				}
				item.Version = version2
				item.Vault.ID = vaultUUID
				item.ID = uuid
				return &item, nil
			}
			Eventually(func() error {
				updatedDeployment := &appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       deploymentKind,
						APIVersion: deploymentAPIVersion,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      deploymentKey.Name,
						Namespace: deploymentKey.Namespace,
						Annotations: map[string]string{
							op.ItemPathAnnotation: itemPath2,
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
				err := k8sClient.Update(ctx, updatedDeployment)
				if err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())

			// TODO: can we achieve the same without sleep?
			time.Sleep(time.Millisecond * 10)
			By("Reading updated K8s secret")
			updatedSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretKey, updatedSecret)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(updatedSecret.Data).Should(Equal(expectedSecretData2))
		})

		It("Should not update if Annotations have not changed", func() {
			deployment1 := &appsv1.Deployment{}
			k8sClient.Get(ctx, deploymentKey, deployment1)

			By("Updating secret without changing annotations")
			Eventually(func() error {
				updatedDeployment := &appsv1.Deployment{
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
				err := k8sClient.Update(ctx, updatedDeployment)
				if err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())

			deployment2 := &appsv1.Deployment{}
			k8sClient.Get(ctx, deploymentKey, deployment2)

			// TODO: can we achieve the same without sleep?
			time.Sleep(time.Millisecond * 10)
			By("Reading updated K8s secret")
			updatedSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, secretKey, updatedSecret)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(updatedSecret.Data).Should(Equal(expectedSecretData))
		})

		It("Should not delete secret created via deployment if it's used in another container", func() {
			By("Create another POD with created secret")
			anotherDeploymentKey := types.NamespacedName{
				Name:      "other-deployment",
				Namespace: namespace,
			}
			Eventually(func() error {
				anotherDeployment := &appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       deploymentKind,
						APIVersion: deploymentAPIVersion,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      anotherDeploymentKey.Name,
						Namespace: anotherDeploymentKey.Namespace,
					},
					Spec: appsv1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": anotherDeploymentKey.Name},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									{
										Name:            anotherDeploymentKey.Name,
										Image:           "eu.gcr.io/kyma-project/example/http-db-service:0.0.6",
										ImagePullPolicy: "IfNotPresent",
										Env: []v1.EnvVar{
											{
												Name: anotherDeploymentKey.Name,
												ValueFrom: &v1.EnvVarSource{
													SecretKeyRef: &v1.SecretKeySelector{
														LocalObjectReference: v1.LocalObjectReference{
															Name: secretKey.Name,
														},
														Key: "password",
													},
												},
											},
										},
									},
								},
							},
						},
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": anotherDeploymentKey.Name},
						},
					},
				}
				err := k8sClient.Create(ctx, anotherDeployment)
				if err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())

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
				f := &v1.Secret{}
				return k8sClient.Get(ctx, secretKey, f)
			}, timeout, interval).Should(Succeed())
		})

		It("Should not delete secret created via deployment if it's used in another volume", func() {
			By("Create another POD with created secret")
			anotherDeploymentKey := types.NamespacedName{
				Name:      "other-deployment",
				Namespace: namespace,
			}
			Eventually(func() error {
				anotherDeployment := &appsv1.Deployment{
					TypeMeta: metav1.TypeMeta{
						Kind:       deploymentKind,
						APIVersion: deploymentAPIVersion,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      anotherDeploymentKey.Name,
						Namespace: anotherDeploymentKey.Namespace,
					},
					Spec: appsv1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: metav1.ObjectMeta{
								Labels: map[string]string{"app": anotherDeploymentKey.Name},
							},
							Spec: v1.PodSpec{
								Volumes: []v1.Volume{
									{
										Name: anotherDeploymentKey.Name,
										VolumeSource: v1.VolumeSource{
											Secret: &v1.SecretVolumeSource{
												SecretName: secretKey.Name,
											},
										},
									},
								},
								Containers: []v1.Container{
									{
										Name:            anotherDeploymentKey.Name,
										Image:           "eu.gcr.io/kyma-project/example/http-db-service:0.0.6",
										ImagePullPolicy: "IfNotPresent",
									},
								},
							},
						},
						Selector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": anotherDeploymentKey.Name},
						},
					},
				}
				err := k8sClient.Create(ctx, anotherDeployment)
				if err != nil {
					return err
				}
				return nil
			}, timeout, interval).Should(Succeed())

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
				f := &v1.Secret{}
				return k8sClient.Get(ctx, secretKey, f)
			}, timeout, interval).Should(Succeed())
		})
	})
})
