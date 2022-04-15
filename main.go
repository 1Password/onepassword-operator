/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/1Password/connect-sdk-go/connect"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"
	"github.com/1Password/onepassword-operator/pkg/utils"
	"github.com/1Password/onepassword-operator/version"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	//	sdkVersion "github.com/operator-framework/operator-sdk/version"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	onepasswordv1 "github.com/1Password/onepassword-operator/api/v1"
	"github.com/1Password/onepassword-operator/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme               = k8sruntime.NewScheme()
	setupLog             = ctrl.Log.WithName("setup")
	WatchNamespaceEnvVar = "WATCH_NAMESPACE"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(onepasswordv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func printVersion() {
	setupLog.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	setupLog.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	setupLog.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	// TODO figure out how to get operator-sdk version
	// setupLog.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
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

	namespace := os.Getenv(WatchNamespaceEnvVar)

	options := ctrl.Options{
		Scheme:                 scheme,
		Namespace:              namespace,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "c26807fd.onepassword.com",
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	if strings.Contains(namespace, ",") {
		options.Namespace = ""
		options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(namespace, ","))
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup One Password Client
	opConnectClient, err := connect.NewClientFromEnvironment()
	if err != nil {
		setupLog.Error(err, "failed to create 1Password client")
		os.Exit(1)
	}

	if err = (&controllers.OnePasswordItemReconciler{
		Client:          mgr.GetClient(),
		Scheme:          mgr.GetScheme(),
		OpConnectClient: opConnectClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OnePasswordItem")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

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

	deploymentNamespace, err := utils.GetOperatorNamespace()
	if err != nil {
		setupLog.Error(err, "Failed to get namespace")
		os.Exit(1)
	}

	//Setup 1PasswordConnect
	if shouldManageConnect() {
		setupLog.Info("Automated Connect Management Enabled")
		go func() {
			connectStarted := false
			for !connectStarted {
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

	// TODO: Configure Metrics Service. See: https://sdk.operatorframework.io/docs/building-operators/golang/migration/#export-metrics

	// Setup update secrets task
	updatedSecretsPoller := op.NewManager(mgr.GetClient(), opConnectClient, shouldAutoRestartDeployments())
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

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Manager exited non-zero")
		done <- true
		os.Exit(1)
	}

}

const manageConnect = "MANAGE_CONNECT"

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

const envPollingIntervalVariable = "POLLING_INTERVAL"
const defaultPollingInterval = 600

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

const restartDeploymentsEnvVariable = "AUTO_RESTART"

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
