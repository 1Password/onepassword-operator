package onepassword

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func IsDeploymentUsingSecrets(deployment *appsv1.Deployment, secrets map[string]*corev1.Secret) bool {
	volumes := deployment.Spec.Template.Spec.Volumes
	containers := deployment.Spec.Template.Spec.Containers
	containers = append(containers, deployment.Spec.Template.Spec.InitContainers...)
	return AreAnnotationsUsingSecrets(deployment.Annotations, secrets) ||
		AreContainersUsingSecrets(containers, secrets) ||
		AreVolumesUsingSecrets(volumes, secrets) ||
		AreImagePullSecretsUsingSecrets(deployment.Spec.Template.Spec.ImagePullSecrets, secrets)
}

func AreImagePullSecretsUsingSecrets(refs []corev1.LocalObjectReference, secrets map[string]*corev1.Secret) bool {
	for _, ref := range refs {
		if _, ok := secrets[ref.Name]; ok {
			return true
		}
	}
	return false
}

func AppendUpdatedImagePullSecrets(
	refs []corev1.LocalObjectReference,
	secrets map[string]*corev1.Secret,
	updatedSecrets map[string]*corev1.Secret,
) {
	for _, ref := range refs {
		if secret, ok := secrets[ref.Name]; ok {
			updatedSecrets[secret.Name] = secret
		}
	}
}
