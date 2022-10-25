package controllers

import (
	"context"
	"math/rand"
	"time"

	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/1Password/onepassword-operator/pkg/mocks"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	onepasswordv1 "github.com/1Password/onepassword-operator/api/v1"
)

const (
	firstHost = "http://localhost:8080"
	awsKey    = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
	iceCream  = "freezing blue 20%"
)

func randomInt() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(1000)
}

var _ = Describe("OnePasswordItem controller", func() {
	BeforeEach(func() {
		// failed test runs that don't clean up leave resources behind.
		err := k8sClient.DeleteAllOf(context.Background(), &onepasswordv1.OnePasswordItem{}, client.InNamespace(namespace))
		Expect(err).ToNot(HaveOccurred())
		err = k8sClient.DeleteAllOf(context.Background(), &v1.Secret{}, client.InNamespace(namespace))
		Expect(err).ToNot(HaveOccurred())

		mocks.DoGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {
			item := onepassword.Item{}
			item.Fields = []*onepassword.ItemField{}
			for k, v := range item1.Data {
				item.Fields = append(item.Fields, &onepassword.ItemField{Label: k, Value: v})
			}
			item.Version = randomInt()
			item.Vault.ID = vaultUUID
			item.ID = uuid
			return &item, nil
		}
	})

	Context("Happy path", func() {
		It("Should handle 1Password Item and secret correctly", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
			}

			key := types.NamespacedName{
				Name:      item1.Name,
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
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Creating the K8s secret successfully")
			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, createdSecret)
				if err != nil {
					return false
				}
				return true
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
			mocks.DoGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {
				item := onepassword.Item{}
				item.Fields = []*onepassword.ItemField{}
				for k, v := range newData {
					item.Fields = append(item.Fields, &onepassword.ItemField{Label: k, Value: v})
				}
				item.Version = randomInt()
				item.Vault.ID = vaultUUID
				item.ID = uuid
				return &item, nil
			}
			_, err := onePasswordItemReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: key})
			Expect(err).ToNot(HaveOccurred())

			updatedSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, updatedSecret)
				if err != nil {
					return false
				}
				return true
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

			mocks.DoGetItemFunc = func(uuid string, vaultUUID string) (*onepassword.Item, error) {
				item := onepassword.Item{}
				item.Title = "!my sECReT it3m%"
				item.Fields = []*onepassword.ItemField{}
				for k, v := range testData {
					item.Fields = append(item.Fields, &onepassword.ItemField{Label: k, Value: v})
				}
				item.Version = randomInt()
				item.Vault.ID = vaultUUID
				item.ID = uuid
				return &item, nil
			}

			By("Creating a new OnePasswordItem successfully")
			Expect(k8sClient.Create(ctx, toCreate)).Should(Succeed())

			created := &onepasswordv1.OnePasswordItem{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, created)
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Creating the K8s secret successfully")
			createdSecret := &v1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(ctx, key, createdSecret)
				if err != nil {
					return false
				}
				return true
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
				Name:      item1.Name,
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
				Name:      "test6",
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
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(secret.Type).Should(Equal(v1.SecretType(customType)))
		})
	})

	Context("Unhappy path", func() {
		It("Should throw an error if K8s Secret type is changed", func() {
			ctx := context.Background()
			spec := onepasswordv1.OnePasswordItemSpec{
				ItemPath: item1.Path,
			}

			key := types.NamespacedName{
				Name:      "test7",
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
				if err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())

			By("Failing to update K8s secret")
			Eventually(func() bool {
				secret.Type = v1.SecretTypeBasicAuth
				err := k8sClient.Update(ctx, secret)
				if err != nil {
					return false
				}
				return true
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
