package main_test

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/marcuscaisey/lox/test/loxtest"
)

func TestClox(t *testing.T) {
	cloxPath := mustBuildClox(t)
	loxtest.TestInterpreter(t, cloxPath)
}

// mustBuildClox build clox and returns the path to it.
func mustBuildClox(t *testing.T) string {
	cmd := exec.CommandContext(t.Context(), "make", "-C", "..")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("building clox: %s: %v\nOutput:\n%s\n", cmd.String(), err, string(output))
	}
	rootDir := loxtest.MustGoModuleRoot(t)
	return filepath.Join(rootDir, "clox/build/release/clox")
}
