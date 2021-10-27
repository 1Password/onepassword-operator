package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	kubeValidate "k8s.io/apimachinery/pkg/util/validation"
)

var invalidDNS1123Chars = regexp.MustCompile("[^a-z0-9-.]+")

func ContainsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

func StringToBool(str string) (bool, error) {
	restartDeploymentBool, err := strconv.ParseBool(strings.ToLower(str))
	if err != nil {
		return false, err
	}
	return restartDeploymentBool, nil
}

// formatSecretName rewrites a value to be a valid Secret name.
//
// The Secret meta.name and data keys must be valid DNS subdomain names
// (https://kubernetes.io/docs/concepts/configuration/secret/#overview-of-secrets)
func FormatSecretName(value string) string {
	if errs := kubeValidate.IsDNS1123Subdomain(value); len(errs) == 0 {
		return value
	}
	return CreateValidSecretName(value)
}

func CreateValidSecretName(value string) string {
	result := strings.ToLower(value)
	result = invalidDNS1123Chars.ReplaceAllString(result, "-")

	if len(result) > kubeValidate.DNS1123SubdomainMaxLength {
		result = result[0:kubeValidate.DNS1123SubdomainMaxLength]
	}

	// first and last character MUST be alphanumeric
	return strings.Trim(result, "-.")
}

func BuildInjectedOnePasswordItemName(vaultId, injectedId string) string {
	return FormatSecretName(fmt.Sprintf("injectedsecret-%s-%s", vaultId, injectedId))
}
