package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Run executes the provided command within this context
func Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	wd, err := os.Getwd()
	if err != nil {
		return wd, err
	}
	wd = strings.Replace(wd, "/test/e2e", "", -1)
	// Command will run from project root
	cmd.Dir = wd

	command := strings.Join(cmd.Args, " ")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return string(output), nil
}
