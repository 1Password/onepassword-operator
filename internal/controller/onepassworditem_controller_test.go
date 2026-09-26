package controller

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	onepasswordv1 "github.com/1Password/onepassword-operator/api/v1"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
)

const (
	firstHost = "http://localhost:8080"
	awsKey    = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	iceCream  = "freezing blue 20%"
)

func findReadyCondition(conditions []onepasswordv1.OnePasswordItemCondition) *onepasswordv1.OnePasswordItemCondition {
	for i := range conditions {
		if conditions[i].Type == onepasswordv1.OnePasswordItemReady {
			return &conditions[i]
		}
	}
	return nil
}

var _ = Describe("OnePasswordItem controller", func() {
	BeforeEach(func() {
		// failed test runs that don't clean up leave resources behind.
		err := k8sClient.DeleteAllOf(context.Background(), &onepasswordv1.OnePasswordItem{}, client.InNamespace(namespace))
		Expect(err).ToNot(HaveOccurred())
		err = k8sClient.DeleteAllOf(context.Background(), &v1.Secret{}, client.InNamespace(namespace))
		Expect(err).ToNot(HaveOccurred())

		item := item1.ToModel()
		mockGetItemByIDFunc.Return(item, nil)
	})

	Context("Happy path", func() {
		It("Should handle 1Password Item and secret correctly", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
			}

			key := types.NamespacedName{
				Name:      "sample-item",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			By("Creating a new OnePasswordItem successfully")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			created := &onepasswordv1.OnePasswordItem{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, created)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating the K8s secret successfully")
			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdSecret.Data).Should(Equal(item1.SecretData))

			By("Updating existing secret successfully")
			newData := map[string]string{
				"username":   "newUser1234",
				"password":   "##newPassword##",
				"extraField": "dev",
			}
			newDataByte := map[string][]byte{
				"username":   []byte("newUser1234"),
				"password":   []byte("##newPassword##"),
				"extraField": []byte("dev"),
			}

			item := item2.ToModel()
			for k, v := range newData {
				item.Fields = append(item.Fields, model.ItemField{Label: k, Value: v})
			}
			mockGetItemByIDFunc.Return(item, nil)

			_, err := onePasswordItemReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).ToNot(HaveOccurred())

			updatedSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, updatedSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(updatedSecret.Data).Should(Equal(newDataByte))

			By("Deleting the OnePasswordItem successfully")
			Eventually(func() error {
				f := &onepasswordv1.OnePasswordItem{}
				err := k8sClient.Get(ctx, key, f)
				if err != nil {
					return err
				}
				return k8sClient.Delete(ctx, f)
			}, timeout, interval).Should(Succeed())

			Eventually(func() error {
				f := &onepasswordv1.OnePasswordItem{}
				return k8sClient.Get(ctx, key, f)
			}, timeout, interval).ShouldNot(Succeed())

			Eventually(func() error {
				f := &v1.Secret{}
				return k8sClient.Get(ctx, key, f)
			}, timeout, interval).ShouldNot(Succeed())
		})

		It("Should handle 1Password Item with fields and sections that have invalid K8s labels correctly", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
			}

			key := types.NamespacedName{
				Name:      "my-secret-it3m",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			testData := map[string]string{
				"username":         username,
				"password":         password,
				"first host":       firstHost,
				"AWS Access Key":   awsKey,
				"😄 ice-cream type": iceCream,
			}
			expectedData := map[string][]byte{
				"username":       []byte(username),
				"password":       []byte(password),
				"first-host":     []byte(firstHost),
				"AWS-Access-Key": []byte(awsKey),
				"ice-cream-type": []byte(iceCream),
			}

			item := item2.ToModel()
			for k, v := range testData {
				item.Fields = append(item.Fields, model.ItemField{Label: k, Value: v})
			}
			mockGetItemByIDFunc.Return(item, nil)

			By("Creating a new OnePasswordItem successfully")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			created := &onepasswordv1.OnePasswordItem{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, created)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating the K8s secret successfully")
			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdSecret.Data).Should(Equal(expectedData))

			By("Deleting the OnePasswordItem successfully")
			Eventually(func() error {
				f := &onepasswordv1.OnePasswordItem{}
				err := k8sClient.Get(ctx, key, f)
				if err != nil {
					return err
				}
				return k8sClient.Delete(ctx, f)
			}, timeout, interval).Should(Succeed())

			Eventually(func() error {
				f := &onepasswordv1.OnePasswordItem{}
				return k8sClient.Get(ctx, key, f)
			}, timeout, interval).ShouldNot(Succeed())

			Eventually(func() error {
				f := &v1.Secret{}
				return k8sClient.Get(ctx, key, f)
			}, timeout, interval).ShouldNot(Succeed())
		})

		It("Should not update K8s secret if OnePasswordItem Version or VaultPath has not changed", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
			}

			key := types.NamespacedName{
				Name:      "item-not-updated",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			By("Creating a new OnePasswordItem successfully")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			item := &onepasswordv1.OnePasswordItem{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, item)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating the K8s secret successfully")
			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(createdSecret.Data).Should(Equal(item1.SecretData))

			By("Updating OnePasswordItem type")
			Eventually(func() bool {
				err1 := k8sClient.Get(ctx, key, item)
				if err1 != nil {
					return false
				}
				item.Type = string(v1.SecretTypeOpaque)
				err := k8sClient.Update(ctx, item)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Reading K8s secret")
			secret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, secret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(secret.Data).Should(Equal(item1.SecretData))
		})

		It("Should rename mapped fields in the generated secret", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
				FieldMapping: map[string]string{
					"password": "myCustomField",
				},
			}

			key := types.NamespacedName{
				Name:      "item-field-mapping",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			expectedData := map[string][]byte{
				"myCustomField": item1.SecretData["password"],
				"username":      item1.SecretData["username"],
			}

			By("Creating a OnePasswordItem with fieldMapping")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			createdSecret := &v1.Secret{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, key, createdSecret)).To(Succeed())
				g.Expect(createdSecret.Data).To(HaveKey("myCustomField"))
				g.Expect(createdSecret.Data).NotTo(HaveKey("password"))
				g.Expect(createdSecret.Data).To(Equal(expectedData))
			}, timeout, interval).Should(Succeed())
		})

		It("Should update the generated secret when fieldMapping changes", func() {
			ctx := context.Background()
			key := types.NamespacedName{
				Name:      "item-field-mapping-update",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: onepasswordv1.OnePasswordItemSpec{
					ItemPath: item1.Path,
				},
			}

			By("Creating a OnePasswordItem without fieldMapping")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			secret := &v1.Secret{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, key, secret)).To(Succeed())
				g.Expect(secret.Data).To(HaveKey("password"))
				g.Expect(secret.Data).NotTo(HaveKey("myCustomField"))
			}, timeout, interval).Should(Succeed())

			By("Adding fieldMapping to the OnePasswordItem")
			Eventually(func() bool {
				item := &onepasswordv1.OnePasswordItem{}
				if err := k8sClient.Get(ctx, key, item); err != nil {
					return false
				}
				item.Spec.FieldMapping = map[string]string{
					"password": "myCustomField",
				}
				return k8sClient.Update(ctx, item) == nil
			}, timeout, interval).Should(BeTrue())

			expectedData := map[string][]byte{
				"myCustomField": item1.SecretData["password"],
				"username":      item1.SecretData["username"],
			}

			By("Updating the K8s Secret data keys")
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, key, secret)).To(Succeed())
				g.Expect(secret.Data).NotTo(HaveKey("password"))
				g.Expect(secret.Data).To(Equal(expectedData))
			}, timeout, interval).Should(Succeed())
		})

		It("Should create custom K8s Secret type using OnePasswordItem", func() {
			const customType = "CustomType"
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
			}

			key := types.NamespacedName{
				Name:      "item-custom-secret-type",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
				Type: customType,
			}

			By("Creating a new OnePasswordItem successfully")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			By("Reading K8s secret")
			secret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, secret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(secret.Type).Should(Equal(v1.SecretType(customType)))
		})

		It("Should handle 1Password Item with a file and populate secret correctly", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
			}

			key := types.NamespacedName{
				Name:      "item-with-file",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			fileContent := []byte("dummy-cert-content")
			item := item1.ToModel()
			item.Files = []model.File{
				{
					ID:          "file-id-123",
					Name:        "server.crt",
					ContentPath: fmt.Sprintf("/v1/vaults/%s/items/%s/files/file-id-123/content", item.VaultID, item.ID),
				},
			}
			item.Files[0].SetContent(fileContent)

			mockGetItemByIDFunc.Return(item, nil)
			mockGetItemByIDFunc.On("GetFileContent", item.VaultID, item.ID, "file-id-123").Return(fileContent, nil)

			By("Creating a new OnePasswordItem with file successfully")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdSecret.Data).Should(HaveKeyWithValue("server.crt", fileContent))
		})
	})

	Context("Unhappy path", func() {
		It("Should set Ready=False when fieldMapping has duplicate targets", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
				FieldMapping: map[string]string{
					"password": "sameKey",
					"username": "sameKey",
				},
			}

			key := types.NamespacedName{
				Name:      "item-invalid-field-mapping",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			By("Creating a OnePasswordItem with invalid fieldMapping")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			created := &onepasswordv1.OnePasswordItem{}
			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Get(ctx, key, created)).To(Succeed())
				condition := findReadyCondition(created.Status.Conditions)
				g.Expect(condition).NotTo(BeNil())
				g.Expect(condition.Status).To(Equal(metav1.ConditionFalse))
				g.Expect(condition.Message).To(ContainSubstring("invalid fieldMapping"))
			}, timeout, interval).Should(Succeed())

			secret := &v1.Secret{}
			Consistently(func() bool {
				return k8sClient.Get(ctx, key, secret) != nil
			}, timeout, interval).Should(BeTrue())
		})

		It("Should throw an error if K8s Secret type is changed", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
			}

			key := types.NamespacedName{
				Name:      "item-changed-secret-type",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			By("Creating a new OnePasswordItem successfully")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			By("Reading K8s secret")
			secret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, secret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Failing to update K8s secret")
			Eventually(func() bool {
				secret.Type = v1.SecretTypeBasicAuth
				err := k8sClient.Update(ctx, secret)
				return err == nil
			}, timeout, interval).Should(BeFalse())
		})

		When("OnePasswordItem resource name contains `_`", func() {
			It("Should fail creating a OnePasswordItem resource", func() {
				ctx := context.Background()
				spec := onepasswordv1.OnePasswordItemSpec{
					ItemPath: item1.Path,
				}

				key := types.NamespacedName{
					Name:      "invalid_name",
					Namespace: namespace,
				}

				toCreate := &onepasswordv1.OnePasswordItem{
					ObjectMeta: metav1.ObjectMeta{
						Name:      key.Name,
						Namespace: key.Namespace,
					},
					Spec: spec,
				}

				By("Creating a new OnePasswordItem")
				Expect(k8sClient.Create(ctx, toCreate)).To(HaveOccurred())

			})
		})

		When("OnePasswordItem resource name contains capital letters", func() {
			It("Should fail creating a OnePasswordItem resource", func() {
				ctx := context.Background()
				spec := onepasswordv1.OnePasswordItemSpec{
					ItemPath: item1.Path,
				}

				key := types.NamespacedName{
					Name:      "invalidName",
					Namespace: namespace,
				}

				toCreate := &onepasswordv1.OnePasswordItem{
					ObjectMeta: metav1.ObjectMeta{
						Name:      key.Name,
						Namespace: key.Namespace,
					},
					Spec: spec,
				}

				By("Creating a new OnePasswordItem")
				Expect(k8sClient.Create(ctx, toCreate)).To(HaveOccurred())
			})
		})
	})
})
