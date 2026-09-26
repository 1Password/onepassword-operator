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

	Context("Template support", func() {
		It("Should create a K8s secret with templated data from a OnePasswordItem", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
				Template: &onepasswordv1.SecretTemplate{
					Data: map[string]string{
						"config.yaml": "user: {{ .Fields.username }}\npass: {{ .Fields.password }}",
					},
				},
			}

			key := types.NamespacedName{
				Name:      "templated-secret",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			By("Creating a new OnePasswordItem with template")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			created := &onepasswordv1.OnePasswordItem{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, created)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("Creating the K8s secret with templated data")
			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			expectedConfig := fmt.Sprintf("user: %s\npass: %s", username, password)
			Expect(createdSecret.Data).Should(HaveKeyWithValue("config.yaml", []byte(expectedConfig)))

			By("Ensuring individual fields are NOT present as separate keys")
			Expect(createdSecret.Data).ShouldNot(HaveKey("username"))
			Expect(createdSecret.Data).ShouldNot(HaveKey("password"))

			By("Deleting the OnePasswordItem successfully")
			Eventually(func() error {
				f := &onepasswordv1.OnePasswordItem{}
				err := k8sClient.Get(ctx, key, f)
				if err != nil {
					return err
				}
				return k8sClient.Delete(ctx, f)
			}, timeout, interval).Should(Succeed())
		})

		It("Should create a K8s secret with multiple templated keys", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
				Template: &onepasswordv1.SecretTemplate{
					Data: map[string]string{
						"DSN":  "postgresql://{{ .Fields.username }}:{{ .Fields.password }}@localhost:5432/mydb",
						"USER": "{{ .Fields.username }}",
					},
				},
			}

			key := types.NamespacedName{
				Name:      "multi-template-secret",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			By("Creating a new OnePasswordItem with multiple template keys")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			expectedDSN := fmt.Sprintf("postgresql://%s:%s@localhost:5432/mydb", username, password)
			Expect(createdSecret.Data).Should(HaveKeyWithValue("DSN", []byte(expectedDSN)))
			Expect(createdSecret.Data).Should(HaveKeyWithValue("USER", []byte(username)))

			By("Deleting the OnePasswordItem successfully")
			Eventually(func() error {
				f := &onepasswordv1.OnePasswordItem{}
				err := k8sClient.Get(ctx, key, f)
				if err != nil {
					return err
				}
				return k8sClient.Delete(ctx, f)
			}, timeout, interval).Should(Succeed())
		})
	})

	Context("ImagePullSecret support", func() {
		It("Should create dockerconfigjson secret from imagePullSecret config", func() {
			ctx := context.Background()
			item := item1.ToModel()
			item.Fields = []model.ItemField{
				{ID: "field-1", Label: "registry", Value: "ghcr.io"},
				{ID: "field-2", Label: "username", Value: "testuser"},
				{ID: "field-3", Label: "password", Value: "testpass"},
				{ID: "field-4", Label: "email", Value: "user@example.com"},
			}
			mockGetItemByIDFunc.Return(item, nil)

			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
				ImagePullSecret: &onepasswordv1.ImagePullSecretConfig{
					RegistryField: "registry",
					UsernameField: "username",
					PasswordField: "password",
					EmailField:    "email",
				},
			}

			key := types.NamespacedName{
				Name:      "ghcr-pull-secret",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
			}

			By("Creating a new OnePasswordItem with imagePullSecret")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			By("Verifying the K8s secret has dockerconfigjson data")
			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdSecret.Data).To(HaveKey(".dockerconfigjson"))
			Expect(createdSecret.Type).To(Equal(v1.SecretTypeDockerConfigJson))

			// Verify the dockerconfigjson contains the registry
			dockerConfigJSON := string(createdSecret.Data[".dockerconfigjson"])
			Expect(dockerConfigJSON).To(ContainSubstring("ghcr.io"))
			Expect(dockerConfigJSON).To(ContainSubstring("testuser"))
		})

		It("Should automatically set secret type to dockerconfigjson when imagePullSecret is configured", func() {
			ctx := context.Background()
			item := item1.ToModel()
			item.Fields = []model.ItemField{
				{ID: "field-1", Label: "registry", Value: "docker.io"},
				{ID: "field-2", Label: "username", Value: "user"},
				{ID: "field-3", Label: "password", Value: "pass"},
			}
			mockGetItemByIDFunc.Return(item, nil)

			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
				ImagePullSecret: &onepasswordv1.ImagePullSecretConfig{
					RegistryField: "registry",
					UsernameField: "username",
					PasswordField: "password",
				},
			}

			key := types.NamespacedName{
				Name:      "auto-dockerconfigjson",
				Namespace: namespace,
			}

			toCreate := &onepasswordv1.OnePasswordItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:      key.Name,
					Namespace: key.Namespace,
				},
				Spec: spec,
				// Type is not explicitly set
			}

			By("Creating a new OnePasswordItem with imagePullSecret but no type")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			By("Verifying the secret type is automatically set to dockerconfigjson")
			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, createdSecret)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			Expect(createdSecret.Type).To(Equal(v1.SecretTypeDockerConfigJson))
		})
	})

	Context("Unhappy path", func() {
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
