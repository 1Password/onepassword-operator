package onepassword

import (
	onepasswordv1 "github.com/1Password/onepassword-operator/operator/pkg/apis/onepassword/v1"
	"github.com/1Password/onepassword-operator/operator/pkg/utils"
	corev1 "k8s.io/api/core/v1"
)

func AreContainersUsingSecrets(containers []corev1.Container, secrets map[string]*corev1.Secret) bool {
	for i := 0; i < len(containers); i++ {
		envVariables := containers[i].Env
		for j := 0; j < len(envVariables); j++ {
			if envVariables[j].ValueFrom != nil && envVariables[j].ValueFrom.SecretKeyRef != nil {
				_, ok := secrets[envVariables[j].ValueFrom.SecretKeyRef.Name]
				if ok {
					return true
				}
			}
		}
	}
	return false
}

func AppendUpdatedContainerSecrets(containers []corev1.Container, secrets map[string]*corev1.Secret, updatedDeploymentSecrets map[string]*corev1.Secret) map[string]*corev1.Secret {
	for i := 0; i < len(containers); i++ {
		envVariables := containers[i].Env
		for j := 0; j < len(envVariables); j++ {
			if envVariables[j].ValueFrom != nil && envVariables[j].ValueFrom.SecretKeyRef != nil {
				secret, ok := secrets[envVariables[j].ValueFrom.SecretKeyRef.Name]
				if ok {
					updatedDeploymentSecrets[secret.Name] = secret
				}
			}
		}
	}
	return updatedDeploymentSecrets
}

func AreContainersUsingInjectedSecrets(containers []corev1.Container, injectedContainers []string, items map[string]*onepasswordv1.OnePasswordItem) bool {
	for _, container := range containers {
		envVariables := container.Env

		// check if container was set to be injected with secrets
		for _, injectedContainer := range injectedContainers {
			if injectedContainer != container.Name {
				continue
			}
		}

		// check if any environment variables are using an updated injected secret
		for _, envVariable := range envVariables {
			referenceVault, referenceItem, err := ParseReference(envVariable.Value)
			if err != nil {
				continue
			}
			_, itemFound := items[utils.BuildInjectedOnePasswordItemName(referenceVault, referenceItem)]
			if itemFound {
				return true
			}
		}
	}
	return false
}
