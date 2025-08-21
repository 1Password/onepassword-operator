package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Run executes the provided command within this context
func Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	rootDir, err := GetProjectRoot()
	if err != nil {
		return "", err
	}

	// Command will run from project root
	cmd.Dir = rootDir

	command := strings.Join(cmd.Args, " ")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%s failed with error: (%v) %s", command, err, string(output))
	}

	return string(output), nil
}

func GetProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// check if go.mod exists in current dir
		modFile := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modFile); err == nil {
			return dir, nil
		}

		// move one level up
		parent := filepath.Dir(dir)
		if parent == dir {
			// reached filesystem root
			return "", fmt.Errorf("project root not found (no go.mod)")
		}
		dir = parent
	}
}
