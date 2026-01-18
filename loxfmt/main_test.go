package main_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/marcuscaisey/lox/test/loxtest"
)

func TestLoxfmt(t *testing.T) {
	rootDir := loxtest.MustGoModuleRoot(t)
	loxfmtPath := loxtest.MustBuildBinary(t, "loxfmt")
	runner := newRunner(rootDir, loxfmtPath)
	loxtest.Run(t, runner)
}

func newRunner(rootDir string, loxfmtPath string) *runner {
	return &runner{
		rootDir:    rootDir,
		loxfmtPath: loxfmtPath,
	}
}

type runner struct {
	rootDir    string
	loxfmtPath string
}

func (r *runner) Test(t *testing.T, path string) {
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	got := r.mustRun(t, path)

	if diff := loxtest.TextDiff(string(got), string(want)); diff != "" {
		t.Errorf("incorrect output printed to stdout:\n%s\nstdout:\n%s", diff, got)
	}
}

func (r *runner) Update(t *testing.T, path string) {
	r.mustRun(t, path, "-write")
}

func (r *runner) mustRun(t *testing.T, path string, flags ...string) string {
	args := append(slices.Clone(flags), path)
	cmd := exec.Command(r.loxfmtPath, args...)

	relPath, err := filepath.Rel(r.rootDir, path)
	if err != nil {
		t.Fatalf("making test file path relative: %s", err)
	}
	logArgs := append(slices.Clone(flags), relPath)
	t.Logf("go run ./loxfmt %s", strings.Join(logArgs, " "))

	stdout, err := cmd.Output()
	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		t.Fatalf("%s\n%s", err, exitErr.Stderr)
	}
	if err != nil {
		t.Fatal(err)
	}

	return string(stdout)
}
