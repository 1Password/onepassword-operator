package onepassword

import corev1 "k8s.io/api/core/v1"

func AreVolumesUsingSecrets(volumes []corev1.Volume, secrets map[string]bool) bool {
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
