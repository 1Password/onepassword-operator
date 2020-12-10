package onepassword

import appsv1 "k8s.io/api/apps/v1"

func IsDeploymentUsingSecrets(deployment *appsv1.Deployment, secrets map[string]bool) bool {
	volumes := deployment.Spec.Template.Spec.Volumes
	containers := deployment.Spec.Template.Spec.Containers
	containers = append(containers, deployment.Spec.Template.Spec.InitContainers...)
	return AreAnnotationsUsingSecrets(deployment.Annotations, secrets) || AreContainersUsingSecrets(containers, secrets) || AreVolumesUsingSecrets(volumes, secrets)
}
