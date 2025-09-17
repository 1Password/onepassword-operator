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
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apix "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/1Password/onepassword-operator/pkg/testhelper/defaults"
)

type TestConfig struct {
	Timeout  time.Duration
	Interval time.Duration
}

type Config struct {
	Namespace    string
	ManifestsDir string
	TestConfig   *TestConfig
	CRDs         []string
}

type Kube struct {
	Config    *Config
	Client    client.Client
	Clientset kubernetes.Interface
	Mapper    meta.RESTMapper
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

	// Install CRDs first (so discovery sees them)
	installCRDs(context.Background(), restConfig, config.CRDs)

	// Build an http.Client from restConfig
	httpClient, err := rest.HTTPClientFor(restConfig)
	Expect(err).NotTo(HaveOccurred())

	// Create a Dynamic RESTMapper that uses restConfig
	rm, err := apiutil.NewDynamicRESTMapper(restConfig, httpClient)
	Expect(err).NotTo(HaveOccurred())

	scheme := runtime.NewScheme()
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))

	kubernetesClient, err := client.New(restConfig, client.Options{
		Scheme: scheme,
		Mapper: rm,
	})
	Expect(err).NotTo(HaveOccurred())

	// Create Kubernetes clientset for logs and other operations
	clientset, err := kubernetes.NewForConfig(restConfig)
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
		Config:    config,
		Client:    kubernetesClient,
		Clientset: clientset,
		Mapper:    rm,
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
		client:    k.Client,
		clientset: k.Clientset,
		config:    k.Config,
		selector:  selector,
	}
}

func (k *Kube) Namespace(name string) *Namespace {
	return &Namespace{
		client: k.Client,
		config: k.Config,
		name:   name,
	}
}

func (k *Kube) Webhook(name string) *Webhook {
	return &Webhook{
		client: k.Client,
		config: k.Config,
		name:   name,
	}
}

// Apply applies a Kubernetes manifest file using server-side apply.
func (k *Kube) Apply(ctx context.Context, fileName string) {
	By("Applying " + fileName)

	// Derive a short-lived context so this API call won't hang indefinitely.
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	data, err := os.ReadFile(k.Config.ManifestsDir + "/" + fileName)
	Expect(err).NotTo(HaveOccurred())

	// Decode YAML -> JSON -> unstructured.Unstructured
	jsonBytes, err := yaml.ToJSON(data)
	Expect(err).NotTo(HaveOccurred())

	var obj unstructured.Unstructured
	Expect(obj.UnmarshalJSON(jsonBytes)).To(Succeed())

	// Default namespace for namespaced resources if not set in YAML
	if obj.GetNamespace() == "" && k.Config.Namespace != "" {
		gvk := obj.GroupVersionKind()
		mapping, mapErr := k.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if mapErr == nil && mapping.Scope.Name() == meta.RESTScopeNameNamespace {
			obj.SetNamespace(k.Config.Namespace)
		}
	}

	// Server-Side Apply (create or update)
	patchOpts := []client.PatchOption{
		client.FieldOwner("onepassword-e2e"),
		client.ForceOwnership, // to force-take conflicting fields
	}
	Expect(k.Client.Patch(c, &obj, client.Apply, patchOpts...)).To(Succeed())
}

func installCRDs(ctx context.Context, restConfig *rest.Config, crdFiles []string) {
	apixClient, err := apix.NewForConfig(restConfig)
	Expect(err).NotTo(HaveOccurred())

	for _, f := range crdFiles {
		By("Installing CRD " + f)
		b, err := os.ReadFile(filepath.Clean(f))
		Expect(err).NotTo(HaveOccurred())

		var crd apiextv1.CustomResourceDefinition
		err = yaml.Unmarshal(b, &crd)
		Expect(err).NotTo(HaveOccurred())

		// Create or Update
		_, err = apixClient.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, &crd, metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			existing, getErr := apixClient.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crd.Name, metav1.GetOptions{})
			Expect(getErr).NotTo(HaveOccurred())

			crd.ResourceVersion = existing.ResourceVersion
			_, err = apixClient.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, &crd, metav1.UpdateOptions{})
		}
		Expect(err).NotTo(HaveOccurred())

		waitCRDEstablished(ctx, apixClient, crd.Name)
	}
}

// waitCRDEstablished Wait until the CRD reaches Established=True, retrying until the suite timeout.
func waitCRDEstablished(ctx context.Context, apixClient *apix.Clientset, name string) {
	By("Waiting for CRD " + name + " to be Established")

	Eventually(func(g Gomega) {
		// Short per-attempt timeout so a single Get can't hang the whole Eventually loop.
		attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		crd, err := apixClient.ApiextensionsV1().
			CustomResourceDefinitions().
			Get(attemptCtx, name, metav1.GetOptions{})
		g.Expect(err).NotTo(HaveOccurred())

		established := false
		for _, c := range crd.Status.Conditions {
			if c.Type == apiextv1.Established && c.Status == apiextv1.ConditionTrue {
				established = true
				break
			}
		}
		g.Expect(established).To(BeTrue(), "CRD %q is not Established yet", name)
	}, defaults.E2ETimeout, defaults.E2EInterval).Should(Succeed())
}
