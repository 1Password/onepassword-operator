package onepassword

import (
	"regexp"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

const AnnotationRegExpString = "^operator.1password.io\\/[a-zA-Z\\.]+"

func TestFilterAnnotations(t *testing.T) {
	invalidAnnotation1 := "onepasswordconnect/vaultId"
	invalidAnnotation2 := "onepasswordconnectkubernetesSecrets"

	annotations := getValidAnnotations()
	annotations[invalidAnnotation1] = "This should be filtered"
	annotations[invalidAnnotation2] = "This should be filtered too"

	r, _ := regexp.Compile(AnnotationRegExpString)
	filteredAnnotations := FilterAnnotations(annotations, r)
	if len(filteredAnnotations) != 2 {
		t.Errorf("Unexpected number of filtered annotations returned. Expected 2, got %v", len(filteredAnnotations))
	}
	_, found := filteredAnnotations[ItemPathAnnotation]
	if !found {
		t.Errorf("One Password Annotation was filtered when it should not have been")
	}
	_, found = filteredAnnotations[NameAnnotation]
	if !found {
		t.Errorf("One Password Annotation was filtered when it should not have been")
	}
}

func TestGetTopLevelAnnotationsForDeployment(t *testing.T) {
	annotations := getValidAnnotations()
	expectedNumAnnotations := len(annotations)
	r, _ := regexp.Compile(AnnotationRegExpString)

	deployment := &appsv1.Deployment{}
	deployment.Annotations = annotations
	filteredAnnotations, annotationsFound := GetAnnotationsForDeployment(deployment, r)

	if !annotationsFound {
		t.Errorf("No annotations marked as found")
	}

	numAnnotations := len(filteredAnnotations)
	if expectedNumAnnotations != numAnnotations {
		t.Errorf("Expected %v annotations got %v", expectedNumAnnotations, numAnnotations)
	}
}

func TestGetTemplateAnnotationsForDeployment(t *testing.T) {
	annotations := getValidAnnotations()
	expectedNumAnnotations := len(annotations)
	r, _ := regexp.Compile(AnnotationRegExpString)

	deployment := &appsv1.Deployment{}
	deployment.Spec.Template.Annotations = annotations
	filteredAnnotations, annotationsFound := GetAnnotationsForDeployment(deployment, r)

	if !annotationsFound {
		t.Errorf("No annotations marked as found")
	}

	numAnnotations := len(filteredAnnotations)
	if expectedNumAnnotations != numAnnotations {
		t.Errorf("Expected %v annotations got %v", expectedNumAnnotations, numAnnotations)
	}
}

func TestGetNoAnnotationsForDeployment(t *testing.T) {
	deployment := &appsv1.Deployment{}
	r, _ := regexp.Compile(AnnotationRegExpString)
	filteredAnnotations, annotationsFound := GetAnnotationsForDeployment(deployment, r)

	if annotationsFound {
		t.Errorf("No annotations should be found")
	}

	numAnnotations := len(filteredAnnotations)
	if numAnnotations != 0 {
		t.Errorf("Expected %v annotations got %v", 0, numAnnotations)
	}
}

func getValidAnnotations() map[string]string {
	return map[string]string{
		ItemPathAnnotation: "vaults/b3e4c7fc-8bf7-4c22-b8bb-147539f10e4f/items/b3e4c7fc-8bf7-4c22-b8bb-147539f10e4f",
		NameAnnotation:     "secretName",
	}
}
