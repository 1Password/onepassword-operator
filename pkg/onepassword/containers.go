package onepassword

import (
	corev1 "k8s.io/api/core/v1"
)

func AreContainersUsingSecrets(containers []corev1.Container, secrets map[string]*corev1.Secret) bool {
	for i := 0; i < len(containers); i++ {
		envVariables := containers[i].Env
		envVariableNames := map[string]struct{}{}

		for j := 0; j < len(envVariables); j++ {
			envVariableNames[envVariables[j].Name] = struct{}{}
			if envVariables[j].ValueFrom != nil && envVariables[j].ValueFrom.SecretKeyRef != nil {
				_, ok := secrets[envVariables[j].ValueFrom.SecretKeyRef.Name]
				if ok {
					return true
				}
			}
		}
		envFromVariables := containers[i].EnvFrom
		for j := 0; j < len(envFromVariables); j++ {
			if envFromVariables[j].SecretRef != nil {
				// Skip env variables that will be overwritten by Env
				if _, ok := envVariableNames[envFromVariables[i].SecretRef.Name]; ok {
					continue;
				}
				_, ok := secrets[envFromVariables[j].SecretRef.Name]
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
		envVariableNames := map[string]struct{}{}

		for j := 0; j < len(envVariables); j++ {
			envVariableNames[envVariables[j].Name] = struct{}{}
			if envVariables[j].ValueFrom != nil && envVariables[j].ValueFrom.SecretKeyRef != nil {
				secret, ok := secrets[envVariables[j].ValueFrom.SecretKeyRef.Name]
				if ok {
					updatedDeploymentSecrets[secret.Name] = secret
				}
			}
		}
		envFromVariables := containers[i].EnvFrom
		for j := 0; j < len(envFromVariables); j++ {
			if envFromVariables[j].SecretRef != nil {
				// Skip env variables that will be overwritten by Env
				if _, ok := envVariableNames[envFromVariables[i].SecretRef.Name]; ok {
					continue;
				}
				secret, ok := secrets[envFromVariables[j].SecretRef.LocalObjectReference.Name]
				if ok {
					updatedDeploymentSecrets[secret.Name] = secret
				}
			}
		}
	}
	return updatedDeploymentSecrets
}
