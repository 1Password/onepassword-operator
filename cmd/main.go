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

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	onepasswordcomv1 "github.com/1Password/onepassword-operator/api/v1"
	"github.com/1Password/onepassword-operator/internal/controller"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"
	opclient "github.com/1Password/onepassword-operator/pkg/onepassword/client"
	"github.com/1Password/onepassword-operator/pkg/utils"
	"github.com/1Password/onepassword-operator/version"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = k8sruntime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

const (
	envPollingIntervalVariable    = "POLLING_INTERVAL"
	manageConnect                 = "MANAGE_CONNECT"
	restartDeploymentsEnvVariable = "AUTO_RESTART"
	defaultPollingInterval        = 600

	annotationRegExpString = "^operator.1password.io\\/[a-zA-Z\\.]+"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

func printVersion() {
	setupLog.Info(fmt.Sprintf("Operator Version: %s", version.OperatorVersion))
	setupLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	setupLog.Info(fmt.Sprintf("Version of operator-sdk: %v", version.OperatorSDKVersion))
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(onepasswordcomv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	printVersion()

	watchNamespace, err := getWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get WatchNamespace, "+
			"the manager will watch and manage resources in all namespaces")
	}

	deploymentNamespace, err := utils.GetOperatorNamespace()
	if err != nil {
		setupLog.Error(err, "Failed to get namespace")
		os.Exit(1)
	}

	options := ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "c26807fd.onepassword.com",
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	if watchNamespace != "" {
		namespaces := strings.Split(watchNamespace, ",")
		namespaceMap := make(map[string]cache.Config)
		for _, namespace := range namespaces {
			namespaceMap[namespace] = cache.Config{}
		}
		options.NewCache = func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			opts.DefaultNamespaces = namespaceMap
			return cache.New(config, opts)
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup One Password Client
	opClient, err := opclient.NewFromEnvironment(opclient.Config{
		Logger:  setupLog,
		Version: version.OperatorVersion,
	})
	if err != nil {
		setupLog.Error(err, "unable to create 1Password client")
		os.Exit(1)
	}

	if err = (&controller.OnePasswordItemReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		OpClient: opClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OnePasswordItem")
		os.Exit(1)
	}

	r, _ := regexp.Compile(annotationRegExpString)
	if err = (&controller.DeploymentReconciler{
		Client:             mgr.GetClient(),
		Scheme:             mgr.GetScheme(),
		OpClient:           opClient,
		OpAnnotationRegExp: r,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Deployment")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	//Setup 1PasswordConnect
	if shouldManageConnect() {
		setupLog.Info("Automated Connect Management Enabled")
		go func() {
			connectStarted := false
			for connectStarted == false {
				err := op.SetupConnect(mgr.GetClient(), deploymentNamespace)
				// Cache Not Started is an acceptable error. Retry until cache is started.
				if err != nil && !errors.Is(err, &cache.ErrCacheNotStarted{}) {
					setupLog.Error(err, "")
					os.Exit(1)
				}
				if err == nil {
					connectStarted = true
				}
			}
		}()
	} else {
		setupLog.Info("Automated Connect Management Disabled")
	}

	// Setup update secrets task
	updatedSecretsPoller := op.NewManager(mgr.GetClient(), opClient, shouldAutoRestartDeployments())
	done := make(chan bool)
	ticker := time.NewTicker(getPollingIntervalForUpdatingSecrets())
	go func() {
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				err := updatedSecretsPoller.UpdateKubernetesSecretsTask()
				if err != nil {
					setupLog.Error(err, "error running update kubernetes secret task")
				}
			}
		}
	}()

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	var watchNamespaceEnvVar = "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

func shouldManageConnect() bool {
	shouldManageConnect, found := os.LookupEnv(manageConnect)
	if found {
		shouldManageConnectBool, err := strconv.ParseBool(strings.ToLower(shouldManageConnect))
		if err != nil {
			setupLog.Error(err, "")
			os.Exit(1)
		}
		return shouldManageConnectBool
	}
	return false
}

func shouldAutoRestartDeployments() bool {
	shouldAutoRestartDeployments, found := os.LookupEnv(restartDeploymentsEnvVariable)
	if found {
		shouldAutoRestartDeploymentsBool, err := strconv.ParseBool(strings.ToLower(shouldAutoRestartDeployments))
		if err != nil {
			setupLog.Error(err, "")
			os.Exit(1)
		}
		return shouldAutoRestartDeploymentsBool
	}
	return false
}

func getPollingIntervalForUpdatingSecrets() time.Duration {
	timeInSecondsString, found := os.LookupEnv(envPollingIntervalVariable)
	if found {
		timeInSeconds, err := strconv.Atoi(timeInSecondsString)
		if err == nil {
			return time.Duration(timeInSeconds) * time.Second
		}
		setupLog.Info("Invalid value set for polling interval. Must be a valid integer.")
	}

	setupLog.Info(fmt.Sprintf("Using default polling interval of %v seconds", defaultPollingInterval))
	return time.Duration(defaultPollingInterval) * time.Second
}
