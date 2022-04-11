package utils

import (
	"strconv"
	"strings"
)

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
