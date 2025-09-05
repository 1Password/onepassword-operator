package system

import (
	"errors"
	"fmt"
	"io"
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

func ReplaceFile(src, dst string) error {
	rootDir, err := GetProjectRoot()
	if err != nil {
		return err
	}

	// Open the source file
	sourceFile, err := os.Open(filepath.Join(rootDir, src))
	if err != nil {
		return err
	}
	defer func(sourceFile *os.File) {
		cerr := sourceFile.Close()
		if err != nil {
			err = errors.Join(err, cerr)
		}
	}(sourceFile)

	// Create (or overwrite) the destination file
	destFile, err := os.Create(filepath.Join(rootDir, dst))
	if err != nil {
		return err
	}
	defer func(destFile *os.File) {
		cerr := destFile.Close()
		if err != nil {
			err = errors.Join(err, cerr)
		}
	}(destFile)

	// Copy contents
	if _, err = io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Ensure data is written to disk
	return destFile.Sync()
}
