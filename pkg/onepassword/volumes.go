package onepassword

import corev1 "k8s.io/api/core/v1"

func AreVolumesUsingSecrets(volumes []corev1.Volume, secrets map[string]*corev1.Secret) bool {
	for i := 0; i < len(volumes); i++ {
		secret := IsVolumeUsingSecret(volumes[i], secrets)
		secretProjection := IsVolumeUsingSecretProjection(volumes[i], secrets)
		if secret == nil && secretProjection == nil {
			return false
		}
	}
	if len(volumes) == 0 {
		return false
	}
	return true
}

func AppendUpdatedVolumeSecrets(volumes []corev1.Volume, secrets map[string]*corev1.Secret, updatedDeploymentSecrets map[string]*corev1.Secret) map[string]*corev1.Secret {
	for i := 0; i < len(volumes); i++ {
		secret := IsVolumeUsingSecret(volumes[i], secrets)
		if secret != nil {
			updatedDeploymentSecrets[secret.Name] = secret
		} else {
			secretProjection := IsVolumeUsingSecretProjection(volumes[i], secrets)
			if secretProjection != nil {
				updatedDeploymentSecrets[secretProjection.Name] = secretProjection
			}
		}
	}
	return updatedDeploymentSecrets
}

func IsVolumeUsingSecret(volume corev1.Volume, secrets map[string]*corev1.Secret) *corev1.Secret {
	if secret := volume.Secret; secret != nil {
		secretName := secret.SecretName
		secretFound, ok := secrets[secretName]
		if ok {
			return secretFound
		}
	}
	return nil
}

func IsVolumeUsingSecretProjection(volume corev1.Volume, secrets map[string]*corev1.Secret) *corev1.Secret {
	if volume.Projected != nil {
		for i := 0; i < len(volume.Projected.Sources); i++ {
			if secret := volume.Projected.Sources[i].Secret; secret != nil {
				secretName := secret.Name
				secretFound, ok := secrets[secretName]
				if ok {
					return secretFound
				}
			}
		}
	}
	return nil
}
