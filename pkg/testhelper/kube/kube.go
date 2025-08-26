package kube

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	apiv1 "github.com/1Password/onepassword-operator/api/v1"
)

type TestConfig struct {
	Timeout  time.Duration
	Interval time.Duration
}

type Config struct {
	Namespace    string
	ManifestsDir string
	TestConfig   *TestConfig
}

type Kube struct {
	Config *Config
	Client client.Client
}

func NewKubeClient(config *Config) *Kube {
	By("Creating a kubernetes client")
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, _ := os.UserHomeDir()
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	restConfig, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	Expect(err).NotTo(HaveOccurred())

	scheme := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(apiv1.AddToScheme(scheme))

	kubernetesClient, err := client.New(restConfig, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())

	// update the current context’s namespace in kubeconfig
	pathOpts := clientcmd.NewDefaultPathOptions()
	cfg, err := pathOpts.GetStartingConfig()
	Expect(err).NotTo(HaveOccurred())

	currentContext := cfg.CurrentContext
	Expect(currentContext).NotTo(BeEmpty(), "no current kube context is set in kubeconfig")

	ctx, ok := cfg.Contexts[currentContext]
	Expect(ok).To(BeTrue(), fmt.Sprintf("current context %q not found in kubeconfig", currentContext))

	ctx.Namespace = config.Namespace
	err = clientcmd.ModifyConfig(pathOpts, *cfg, true)
	Expect(err).NotTo(HaveOccurred())

	return &Kube{
		Config: config,
		Client: kubernetesClient,
	}
}

func (k *Kube) Secret(name string) *Secret {
	return &Secret{
		client: k.Client,
		config: k.Config,
		name:   name,
	}
}

func (k *Kube) Deployment(name string) *Deployment {
	return &Deployment{
		client: k.Client,
		config: k.Config,
		name:   name,
	}
}

func (k *Kube) Pod(selector map[string]string) *Pod {
	return &Pod{
		client:   k.Client,
		config:   k.Config,
		selector: selector,
	}
}

// ApplyOnePasswordItem applies a OnePasswordItem manifest.
func (k *Kube) ApplyOnePasswordItem(ctx context.Context, fileName string) {
	By("Applying " + fileName)

	// Derive a short-lived context so this API call won't hang indefinitely.
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	data, err := os.ReadFile(k.Config.ManifestsDir + "/" + fileName)
	Expect(err).NotTo(HaveOccurred())

	item := &apiv1.OnePasswordItem{}
	err = yaml.Unmarshal(data, item)
	Expect(err).NotTo(HaveOccurred())

	if item.Namespace == "" {
		item.Namespace = k.Config.Namespace
	}

	err = k.Client.Get(c, client.ObjectKey{Name: item.Name, Namespace: k.Config.Namespace}, item)
	if errors.IsNotFound(err) {
		err = k.Client.Create(c, item)
	} else {
		err = k.Client.Update(c, item)
	}
	Expect(err).NotTo(HaveOccurred())
}
