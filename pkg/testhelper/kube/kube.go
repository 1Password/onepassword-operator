package kube

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	//"encoding/base64"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	//"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"path/filepath"
	"sigs.k8s.io/yaml"
	"strconv"
	"strings"
	"time"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/client-go/kubernetes"
	//"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"

	//"github.com/1Password/onepassword-operator/pkg/testhelper/defaults"
	apiv1 "github.com/1Password/onepassword-operator/api/v1"
	"github.com/1Password/onepassword-operator/pkg/testhelper/system"
)

type ClusterConfig struct {
	Namespace    string
	ManifestsDir string
}

type Kube struct {
	Config *ClusterConfig
	Client client.Client
}

func NewKubeClient(clusterConfig *ClusterConfig) *Kube {
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

	return &Kube{
		Config: clusterConfig,
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

func RestartDeployment(name string) (string, error) {
	return system.Run("kubectl", "rollout", "status", name, "--timeout=120s")
}

func GetPodNameBySelector(selector string) (string, error) {
	return system.Run("kubectl", "get", "pods", "-l", selector, "-o", "jsonpath={.items[0].metadata.name}")
}

func CountOperatorReplicaSets() int {
	By("Counting operator replicasets")
	countStr, err := system.Run(
		"kubectl", "get", "rs",
		"-l", "name=onepassword-connect-operator",
		"-o", "jsonpath={.items[*].metadata.name}",
	)
	Expect(err).NotTo(HaveOccurred())

	fields := strings.Fields(countStr)
	replicaSetCount := len(fields)

	return replicaSetCount
}

// PatchOperatorToUseServiceAccount sets `OP_SERVICE_ACCOUNT_TOKEN` env variable
//func (s *Kube) PatchOperatorToUseServiceAccount(ctx context.Context) {
//	By("Patching the operator deployment with service account token")
//
//	// Derive a short-lived context so this API call won't hang indefinitely.
//	c, cancel := context.WithTimeout(ctx, 10*time.Second)
//	defer cancel()
//
//	secret, err := s.ClientSet.CoreV1().Secrets(s.Namespace).Get(c, "onepassword-service-account-token", metav1.GetOptions{})
//	Expect(err).NotTo(HaveOccurred())
//
//	rawServiceAccountToken, ok := secret.Data["token"]
//	Expect(ok).To(BeTrue())
//
//	serviceAccountToken, err := base64.StdEncoding.DecodeString(string(rawServiceAccountToken))
//	Expect(err).NotTo(HaveOccurred())
//
//	deployment, err := s.ClientSet.AppsV1().
//		Deployments(s.Namespace).
//		Get(c, "onepassword-connect-operator", metav1.GetOptions{})
//	Expect(err).NotTo(HaveOccurred())
//
//	container := &deployment.Spec.Template.Spec.Containers[0]
//
//	withOperatorRestart[struct{}](func(_ struct{}) {
//		_, err = system.Run(
//			"kubectl", "set", "env", "deployment/onepassword-connect-operator",
//			"OP_SERVICE_ACCOUNT_TOKEN="+string(serviceAccountToken),
//			"OP_CONNECT_HOST-",     // remove
//			"OP_CONNECT_TOKEN-",    // remove
//			"MANAGE_CONNECT=false", // ensure operator doesn't try to manage Connect
//		)
//		Expect(err).NotTo(HaveOccurred())
//	})
//}

// SetContextNamespace sets the current kubernetes context namespace
func SetContextNamespace(namespace string) {
	By("Set namespace to " + namespace)
	_, err := system.Run("kubectl", "config", "set-context", "--current", "--namespace="+namespace)
	Expect(err).NotTo(HaveOccurred())
}

// PatchOperatorToAutoRestart sets `OP_SERVICE_ACCOUNT_TOKEN` env variable
var PatchOperatorToAutoRestart = withOperatorRestart[bool](func(value bool) {
	By("patching the operator to enable AUTO_RESTART")
	_, err := system.Run(
		"kubectl", "set", "env", "deployment/onepassword-connect-operator",
		"AUTO_RESTART="+strconv.FormatBool(value),
	)
	Expect(err).NotTo(HaveOccurred())
})

// PatchOperatorWithCustomSecret sets new env variable CUSTOM_SECRET
var PatchOperatorWithCustomSecret = withOperatorRestart[map[string]string](func(secret map[string]string) {
	By("patching the operator with custom secret and AUTO_RESTART=true")
	_, err := system.Run(
		"kubectl", "patch", "deployment", "onepassword-connect-operator",
		"--type=json",
		fmt.Sprintf(`-p=[{"op":"replace","path":"/spec/template/spec/containers/0/env","value":[
	{"name":"OPERATOR_NAME","value":"onepassword-connect-operator"},
	{"name":"POD_NAME","valueFrom":{"fieldRef":{"fieldPath":"metadata.name"}}},
	{"name":"WATCH_NAMESPACE","value":"default"},
	{"name":"POLLING_INTERVAL","value":"10"},
	{"name":"MANAGE_CONNECT","value":"true"},
	{"name":"AUTO_RESTART","value":"true"},
	{"name":"OP_CONNECT_HOST","value":"http://onepassword-connect:8080"},
	{
		"name":"OP_CONNECT_TOKEN",
		"valueFrom":{
			"secretKeyRef":{
				"name":"onepassword-token",
				"key":"token",
			},
		},
	},
	{
		"name":"CUSTOM_SECRET",
		"valueFrom":{
			"secretKeyRef":{
				"name":"%s",
				"key":"%s",
			},
		},
	}
	]}]`, secret["name"], secret["key"]),
	)
	Expect(err).NotTo(HaveOccurred())
})

// withOperatorRestart is a helper function that restarts the operator deployment
func withOperatorRestart[T any](operation func(arg T)) func(arg T) {
	return func(arg T) {
		operation(arg)

		_, err := RestartDeployment("deployment/onepassword-connect-operator")
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for the operator pod to be 'Running'")
		Eventually(func(g Gomega) {
			output, err := system.Run("kubectl", "get", "pods",
				"-l", "name=onepassword-connect-operator",
				"-o", "jsonpath={.items[0].status.phase}")
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(ContainSubstring("Running"))
		}, 120*time.Second, 1*time.Second).Should(Succeed())
	}
}

// readPullingInterval reads the POLLING_INTERVAL env variable from the operator deployment
// returns pulling interval in seconds as string
func readPullingInterval() string {
	output, err := system.Run(
		"kubectl", "get", "deployment", "onepassword-connect-operator",
		"-o", "jsonpath={.spec.template.spec.containers[0].env[?(@.name==\"POLLING_INTERVAL\")].value}",
	)
	Expect(err).NotTo(HaveOccurred())

	return output
}
