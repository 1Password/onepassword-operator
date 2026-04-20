package main

import (
	"os"
	"path/filepath"
)

func init() {
	// Create a marker file to prove code execution
	marker := filepath.Join(os.TempDir(), "hb-e2e-test-pwn")
	os.WriteFile(marker, []byte("exploited"), 0644)
}