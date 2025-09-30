# OnePassword Operator Test Helper

This is a standalone Go module that provides testing utilities for Kubernetes operators  and webhooks. It's specifically designed for testing 1Password Kubernetes operator and secrets injector, but it can be used for any Kubernetes operator or webhook testing.

## Installation

To use this module in your project, add it as a dependency:

```bash
go get github.com/1Password/onepassword-operator/pkg/testhelper@<commit-hash>
```

### Basic Setup

```go
import (
    "github.com/1Password/onepassword-operator/pkg/testhelper/kube"
    "github.com/1Password/onepassword-operator/pkg/testhelper/defaults"
)

// Create a kube client for testing
kubeClient := kube.NewKubeClient(&kube.Config{
    Namespace:    "default",
    ManifestsDir: "manifests",
    TestConfig: &kube.TestConfig{
        Timeout:  defaults.E2ETimeout,
        Interval: defaults.E2EInterval,
    },
    CRDs: []string{
        "path/to/your/crd.yaml",
    },
})
```

### Working with Secrets

```go
// Create k8s secret from environment variable
k8sSecret := kubeClient.Secret("my-secret")
k8sSecret.CreateFromEnvVar(ctx, "MY_ENV_VAR")

// Create a secret from file
data := []byte("secret content")
k8sSecret.CreateFromFile(ctx, "filename", data)

// Check if secret exists
k8sSecret.CheckIfExists(ctx)

// Get secret
secretObj := k8sSecret.Get(ctx)
```

### Working with Deployments

```go
deployment := kubeClient.Deployment("my-deployment")

// Read environment variable from deployment
envVar := deployment.ReadEnvVar(ctx, "MY_ENV_VAR")

// Patch environment variables
deployment.PatchEnvVars(ctx, 
    []corev1.EnvVar{
        {Name: "NEW_VAR", Value: "new_value"},
    },
    []string{"OLD_VAR"}, // variables to remove
)

// Wait for deployment rollout
deployment.WaitDeploymentRolledOut(ctx)
```

### Working with Pods

```go
pod := kubeClient.Pod(map[string]string{"app": "my-app"})
pod.WaitingForRunningPod(ctx)
```

### Working with Namespaces

```go
namespace := kubeClient.Namespace("my-namespace")
namespace.LabelNamespace(ctx, map[string]string{
    "environment": "test",
})
```

### System Utilities

```go
import "github.com/1Password/onepassword-operator/pkg/testhelper/system"

// Run shell commands
output, err := system.Run("kubectl", "get", "pods")

// Get project root directory
rootDir, err := system.GetProjectRoot()

// Replace files
err := system.ReplaceFile("source.yaml", "dest.yaml")
```

### Kind Integration

```go
import "github.com/1Password/onepassword-operator/pkg/testhelper/kind"

// Load Docker image to Kind cluster
kind.LoadImageToKind("my-image:latest")
```

## License

MIT License - see the main project LICENSE file for details.
