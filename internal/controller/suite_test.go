/*
MIT License

Copyright (c) 2020-2024 1Password

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	onepasswordcomv1 "github.com/1Password/onepassword-operator/api/v1"
	"github.com/1Password/onepassword-operator/pkg/mocks"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

const (
	username = "test-user"
	password = "QmHumKc$mUeEem7caHtbaBaJ"

	username2 = "test-user2"
	password2 = "4zotzqDqXKasLFT2jzTs"

	annotationRegExpString = "^operator\\.1password\\.io\\/[a-zA-Z\\.]+"
)

// Define utility constants for object names and testing timeouts/durations and intervals.
const (
	namespace = "default"

	timeout  = time.Second * 10
	duration = time.Second * 10
	interval = time.Millisecond * 250
)

var (
	cfg                       *rest.Config
	k8sClient                 client.Client
	testEnv                   *envtest.Environment
	ctx                       context.Context
	cancel                    context.CancelFunc
	onePasswordItemReconciler *OnePasswordItemReconciler
	deploymentReconciler      *DeploymentReconciler
	mockGetItemByIDFunc       *mock.Call

	item1 = &TestItem{
		ItemID:  "nwrhuano7bcwddcviubpp4mhfq",
		VaultID: "hfnjvi6aymbsnfc2xeeoheizda",
		Name:    "test-item",
		Version: 123,
		Path:    "vaults/hfnjvi6aymbsnfc2xeeoheizda/items/nwrhuano7bcwddcviubpp4mhfq",
		Data: map[string]string{
			"username": username,
			"password": password,
		},
		SecretData: map[string][]byte{
			"password": []byte(password),
			"username": []byte(username),
		},
	}

	item2 = &TestItem{
		ItemID:  "nwrhuano7bcwddcviubpp4mhf2",
		VaultID: "hfnjvi6aymbsnfc2xeeoheizd2",
		Name:    "test-item2",
		Path:    "vaults/hfnjvi6aymbsnfc2xeeoheizd2/items/nwrhuano7bcwddcviubpp4mhf2",
		Version: 456,
		Data: map[string]string{
			"username": username2,
			"password": password2,
		},
		SecretData: map[string][]byte{
			"password": []byte(password2),
			"username": []byte(username2),
		},
	}
)

type TestItem struct {
	ItemID     string
	VaultID    string
	Name       string
	Version    int
	Path       string
	Data       map[string]string
	SecretData map[string][]byte
}

func (ti *TestItem) ToModel() *model.Item {
	item := &model.Item{}
	item.Version = ti.Version
	item.VaultID = ti.VaultID
	item.ID = ti.ItemID

	item.Fields = []model.ItemField{}
	for k, v := range ti.Data {
		item.Fields = append(item.Fields, model.ItemField{Label: k, Value: v})
	}

	return item
}

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.TODO())

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	// Retrieve the first found binary directory to allow running tests from IDEs
	if getFirstFoundEnvTestBinaryDir() != "" {
		testEnv.BinaryAssetsDirectory = getFirstFoundEnvTestBinaryDir()
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = onepasswordcomv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	mockOpClient := &mocks.TestClient{}
	mockGetItemByIDFunc = mockOpClient.On("GetItemByID", mock.Anything, mock.Anything)

	onePasswordItemReconciler = &OnePasswordItemReconciler{
		Client:   k8sManager.GetClient(),
		Scheme:   k8sManager.GetScheme(),
		OpClient: mockOpClient,
	}
	err = (onePasswordItemReconciler).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	r, _ := regexp.Compile(annotationRegExpString)
	deploymentReconciler = &DeploymentReconciler{
		Client:             k8sManager.GetClient(),
		Scheme:             k8sManager.GetScheme(),
		OpClient:           mockOpClient,
		OpAnnotationRegExp: r,
		Recorder:           k8sManager.GetEventRecorderFor("onepassword-operator-deployment"),
	}
	err = (deploymentReconciler).SetupWithManager(k8sManager)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = k8sManager.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

// getFirstFoundEnvTestBinaryDir locates the first binary in the specified path.
// ENVTEST-based tests depend on specific binaries, usually located in paths set by
// controller-runtime. When running tests directly (e.g., via an IDE) without using
// Makefile targets, the 'BinaryAssetsDirectory' must be explicitly configured.
//
// This function streamlines the process by finding the required binaries, similar to
// setting the 'KUBEBUILDER_ASSETS' environment variable. To ensure the binaries are
// properly set up, run 'make setup-envtest' beforehand.
func getFirstFoundEnvTestBinaryDir() string {
	basePath := filepath.Join("..", "..", "bin", "k8s")
	entries, err := os.ReadDir(basePath)
	if err != nil {
		logf.Log.Error(err, "Failed to read directory", "path", basePath)
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(basePath, entry.Name())
		}
	}
	return ""
}
