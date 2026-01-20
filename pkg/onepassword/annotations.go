package onepassword

import (
	"regexp"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	OnepasswordPrefix             = "operator.1password.io"
	ItemPathAnnotation            = OnepasswordPrefix + "/item-path"
	NameAnnotation                = OnepasswordPrefix + "/item-name"
	VersionAnnotation             = OnepasswordPrefix + "/item-version"
	RestartAnnotation             = OnepasswordPrefix + "/last-restarted"
	AutoRestartWorkloadAnnotation = OnepasswordPrefix + "/auto-restart"
)

func GetAnnotationsForDeployment(deployment *appsv1.Deployment, regex *regexp.Regexp) (map[string]string, bool) {
	annotationsFound := false
	annotations := FilterAnnotations(deployment.Annotations, regex)
	if len(annotations) > 0 {
		annotationsFound = true
	} else {
		annotations = FilterAnnotations(deployment.Spec.Template.Annotations, regex)
		if len(annotations) > 0 {
			annotationsFound = true
		} else {
			annotationsFound = false
		}
	}

	return annotations, annotationsFound
}

func FilterAnnotations(annotations map[string]string, regex *regexp.Regexp) map[string]string {
	filteredAnnotations := make(map[string]string)
	for key, value := range annotations {
		if regex.MatchString(key) && key != RestartAnnotation && key != AutoRestartWorkloadAnnotation {
			filteredAnnotations[key] = value
		}
	}
	return filteredAnnotations
}

func AreAnnotationsUsingSecrets(annotations map[string]string, secrets map[string]*corev1.Secret) bool {
	_, ok := secrets[annotations[NameAnnotation]]
	return ok
}

func AppendAnnotationUpdatedSecret(
	annotations map[string]string,
	secrets map[string]*corev1.Secret,
	updatedDeploymentSecrets map[string]*corev1.Secret,
) map[string]*corev1.Secret {
	secret, ok := secrets[annotations[NameAnnotation]]
	if ok {
		updatedDeploymentSecrets[secret.Name] = secret
	}
	return updatedDeploymentSecrets
}
