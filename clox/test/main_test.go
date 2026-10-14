package main_test

import (
	_ "embed"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcuscaisey/lox/test/loxtest"
)

func TestClox(t *testing.T) {
	cloxPath := mustBuildClox(t)
	runner := newRunner(t, cloxPath)
	loxtest.Run(t, runner, loxtest.WithSkipSyntaxErrors(false))
}

// runner like [loxtest.InterpreterRunner] except it skips tests whose prefix is listed in
// skipped_tests.txt.
type runner struct {
	interpreterRunner loxtest.Runner
}

func newRunner(t *testing.T, interpreterPath string) *runner {
	interpreterRunner := loxtest.NewInterpreterRunner(t, interpreterPath)
	return &runner{
		interpreterRunner: interpreterRunner,
	}
}

func (r *runner) Test(t *testing.T, path string) {
	if shouldSkipTest(t.Name()) {
		t.Skip("test listed in skipped_tests.txt")
	}
	r.interpreterRunner.Test(t, path)
}

func (r *runner) Update(t *testing.T, path string) {
	if shouldSkipTest(t.Name()) {
		t.Skip("test listed in skipped_tests.txt")
	}
	r.interpreterRunner.Update(t, path)
}

//go:embed skipped_tests.txt
var skippedTestsTxt string

var skippedTestPrefixes = func() []string {
	trimmed := strings.TrimSpace(skippedTestsTxt)
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "\n")
}()

func shouldSkipTest(name string) bool {
	for _, prefix := range skippedTestPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
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
