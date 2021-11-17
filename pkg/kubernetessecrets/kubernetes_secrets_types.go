package kubernetessecrets

import (
	corev1 "k8s.io/api/core/v1"
)

// Default to Opaque as this is Kubernetes' default
var KubernetesSecretTypes = map[string]corev1.SecretType{
	"Opaque":                              corev1.SecretTypeOpaque,
	"kubernetes.io/basic-auth":            corev1.SecretTypeBasicAuth,
	"kubernetes.io/service-account-token": corev1.SecretTypeServiceAccountToken,
	"kubernetes.io/dockercfg":             corev1.SecretTypeDockercfg,
	"kubernetes.io/dockerconfigjson":      corev1.SecretTypeDockerConfigJson,
	"kubernetes.io/ssh-auth":              corev1.SecretTypeSSHAuth,
	"kubernetes.io/tls":                   corev1.SecretTypeTLS,
	"bootstrap.kubernetes.io/token":       corev1.SecretTypeBootstrapToken,
}
