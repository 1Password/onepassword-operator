package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/1Password/onepassword-operator/pkg/controller"
	op "github.com/1Password/onepassword-operator/pkg/onepassword"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"

	"github.com/1Password/onepassword-operator/pkg/apis"
	"github.com/1Password/onepassword-operator/version"

	"github.com/1Password/connect-sdk-go/connect"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const envPollingIntervalVariable = "POLLING_INTERVAL"
const manageConnect = "MANAGE_CONNECT"
const restartDeploymentsEnvVariable = "AUTO_RESTART"
const defaultPollingInterval = 600

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)
var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Operator Version: %s", version.Version))
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	namespace := os.Getenv(k8sutil.WatchNamespaceEnvVar)

	deploymentNamespace, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		log.Error(err, "Failed to get namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.Background()
	// Become the leader before proceeding
	err = leader.Become(ctx, "onepassword-connect-operator-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Set default manager options
	options := manager.Options{
		Namespace:          namespace,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	}

	// Add support for MultiNamespace set in WATCH_NAMESPACE (e.g ns1,ns2)
	// Note that this is not intended to be used for excluding namespaces, this is better done via a Predicate
	// Also note that you may face performance issues when using this with a high number of namespaces.
	if strings.Contains(namespace, ",") {
		options.Namespace = ""
		options.NewCache = cache.MultiNamespacedCacheBuilder(strings.Split(namespace, ","))
	}

	// Create a new manager to provide shared dependencies and start components
	mgr, err := manager.New(cfg, options)
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	//Setup 1PasswordConnect
	if shouldManageConnect() {
		log.Info("Automated Connect Management Enabled")
		go func() {
			connectStarted := false
			for connectStarted == false {
				err := op.SetupConnect(mgr.GetClient(), deploymentNamespace)
				// Cache Not Started is an acceptable error. Retry until cache is started.
				if err != nil && !errors.Is(err, &cache.ErrCacheNotStarted{}) {
					log.Error(err, "")
					os.Exit(1)
				}
				if err == nil {
					connectStarted = true
				}
			}
		}()
	} else {
		log.Info("Automated Connect Management Disabled")
	}

	// Setup One Password Client
	opConnectClient, err := connect.NewClientFromEnvironment()

	if err := controller.AddToManager(mgr, opConnectClient); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Add the Metrics Service
	addMetrics(ctx, cfg)

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
					log.Error(err, "error running update kubernetes secret task")
				}
			}
		}
	}()

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		done <- true
		os.Exit(1)
	}
}

// addMetrics will create the Services and Service Monitors to allow the operator export the metrics by using
// the Prometheus operator
func addMetrics(ctx context.Context, cfg *rest.Config) {
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		if errors.Is(err, k8sutil.ErrRunLocal) {
			log.Info("Skipping CR metrics server creation; not running in a cluster.")
			return
		}
	}

	if err := serveCRMetrics(cfg, operatorNs); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}

	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}

	// The ServiceMonitor is created in the same namespace where the operator is deployed
	_, err = metrics.CreateServiceMonitors(cfg, operatorNs, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config, operatorNs string) error {
	// The function below returns a list of filtered operator/CR specific GVKs. For more control, override the GVK list below
	// with your own custom logic. Note that if you are adding third party API schemas, probably you will need to
	// customize this implementation to avoid permissions issues.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}

	// The metrics will be generated from the namespaces which are returned here.
	// NOTE that passing nil or an empty list of namespaces in GenerateAndServeCRMetrics will result in an error.
	ns, err := kubemetrics.GetNamespacesForMetrics(operatorNs)
	if err != nil {
		return err
	}

	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}

func getPollingIntervalForUpdatingSecrets() time.Duration {
	timeInSecondsString, found := os.LookupEnv(envPollingIntervalVariable)
	if found {
		timeInSeconds, err := strconv.Atoi(timeInSecondsString)
		if err == nil {
			return time.Duration(timeInSeconds) * time.Second
		}
		log.Info("Invalid value set for polling interval. Must be a valid integer.")
	}

	log.Info(fmt.Sprintf("Using default polling interval of %v seconds", defaultPollingInterval))
	return time.Duration(defaultPollingInterval) * time.Second
}

func shouldManageConnect() bool {
	shouldManageConnect, found := os.LookupEnv(manageConnect)
	if found {
		shouldManageConnectBool, err := strconv.ParseBool(strings.ToLower(shouldManageConnect))
		if err != nil {
			log.Error(err, "")
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
			log.Error(err, "")
			os.Exit(1)
		}
		return shouldAutoRestartDeploymentsBool
	}
	return false
}
