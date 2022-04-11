package onepassword

import corev1 "k8s.io/api/core/v1"

func AreVolumesUsingSecrets(volumes []corev1.Volume, secrets map[string]*corev1.Secret) bool {
	for i := 0; i < len(volumes); i++ {
		if secret := volumes[i].Secret; secret != nil {
			secretName := secret.SecretName
			_, ok := secrets[secretName]
			if ok {
				return true
			}
		}
	}
	return false
}

func AppendUpdatedVolumeSecrets(volumes []corev1.Volume, secrets map[string]*corev1.Secret, updatedDeploymentSecrets map[string]*corev1.Secret) map[string]*corev1.Secret {
	for i := 0; i < len(volumes); i++ {
		if secret := volumes[i].Secret; secret != nil {
			secretName := secret.SecretName
			secret, ok := secrets[secretName]
			if ok {
				updatedDeploymentSecrets[secret.Name] = secret
			}
		}
	}
	return updatedDeploymentSecrets
}
