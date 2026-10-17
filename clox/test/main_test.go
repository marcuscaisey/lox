package main_test

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/marcuscaisey/lox/test/loxtest"
)

func TestClox(t *testing.T) {
	// We stat all of cloxes sources so that they'll be tracked by go test.
	//
	// Since clox is not written in Go, its sources are not tracked by go test. This means that when
	// a source file changes, go test might output results from its cache.
	// From 'go help test':
	//   Tests that open files within the package's module or that consult environment variables
	//   only match future runs in which the files and environment variables are unchanged.
	// Therefore we just need to stat all of them.
	statCloxSources(t)

	cloxPath := mustBuildClox(t)
	runner := newRunner(t, cloxPath)
	loxtest.Run(t, runner, loxtest.WithSkipSyntaxErrors(false))
}

func statCloxSources(t *testing.T) {
	t.Helper()
	for _, pattern := range []string{"../*.c", "../*.h", "../Makefile"} {
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range files {
			if _, err := os.Stat(file); err != nil {
				t.Fatal(err)
			}
		}
	}
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
