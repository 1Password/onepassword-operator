package kube

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"time"

	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/1Password/onepassword-operator/pkg/testhelper/defaults"
	"github.com/1Password/onepassword-operator/pkg/testhelper/system"
)

type Secret struct {
	client client.Client
	config *ClusterConfig
	name   string
}

// CreateFromEnvVar creates a kubernetes secret from an environment variable
func (s *Secret) CreateFromEnvVar(ctx context.Context, envVar string) *corev1.Secret {
	By("Creating '" + s.name + "' secret from environment variable")

	// Derive a short-lived context so this API call won't hang indefinitely.
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	value, ok := os.LookupEnv(envVar)
	Expect(ok).To(BeTrue())
	Expect(value).NotTo(BeEmpty())

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.name,
			Namespace: s.config.Namespace,
		},
		StringData: map[string]string{
			"token": value,
		},
	}

	err := s.client.Create(c, secret)
	Expect(err).NotTo(HaveOccurred())

	return secret
}

// CreateFromFile creates a kubernetes secret from a file
func (s *Secret) CreateFromFile(ctx context.Context, fileName string, content []byte) *corev1.Secret {
	By("Creating '" + s.name + "' secret from file " + fileName)

	// Derive a short-lived context so this API call won't hang indefinitely.
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.name,
			Namespace: s.config.Namespace,
		},
		Data: map[string][]byte{
			filepath.Base(fileName): content,
		},
	}

	err := s.client.Create(c, secret)
	Expect(err).NotTo(HaveOccurred())

	return secret
}

// CreateOpCredentials creates a kubernetes secret from 1password-credentials.json file in the project root
// encodes it in base64 and saves it to op-session file
func (s *Secret) CreateOpCredentials(ctx context.Context) *corev1.Secret {
	rootDir, err := system.GetProjectRoot()
	Expect(err).NotTo(HaveOccurred())

	credentialsFilePath := filepath.Join(rootDir, "1password-credentials.json")
	data, err := os.ReadFile(credentialsFilePath)
	Expect(err).NotTo(HaveOccurred())

	encoded := base64.RawURLEncoding.EncodeToString(data)

	return s.CreateFromFile(ctx, "op-session", []byte(encoded))
}

// Get retrieves a kubernetes secret
func (s *Secret) Get(ctx context.Context) *corev1.Secret {
	By("Getting '" + s.name + "' secret")

	// Derive a short-lived context so this API call won't hang indefinitely.
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	secret := &corev1.Secret{}
	err := s.client.Get(c, client.ObjectKey{Name: s.name, Namespace: s.config.Namespace}, secret)
	Expect(err).NotTo(HaveOccurred())

	return secret
}

// Delete deletes a kubernetes secret
func (s *Secret) Delete(ctx context.Context) {
	By("Deleting '" + s.name + "' secret")

	// Derive a short-lived context so this API call won't hang indefinitely.
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.name,
			Namespace: s.config.Namespace,
		},
	}
	err := s.client.Delete(c, secret)
	Expect(err).NotTo(HaveOccurred())
}

// CheckIfExists repeatedly attempts to retrieve the given Secret
// from the cluster until it is found or the test's timeout expires.
func (s *Secret) CheckIfExists(ctx context.Context) {
	By("Checking '" + s.name + "' secret")

	Eventually(func(g Gomega) {
		// Derive a short-lived context so this API call won't hang indefinitely.
		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		secret := &corev1.Secret{}
		err := s.client.Get(attemptCtx, client.ObjectKey{Name: s.name, Namespace: s.config.Namespace}, secret)
		g.Expect(err).NotTo(HaveOccurred())
	}, defaults.E2ETimeout, defaults.E2EInterval).Should(Succeed())
}
