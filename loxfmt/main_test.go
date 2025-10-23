package main_test

import (
	"errors"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/marcuscaisey/lox/test/loxtest"
)

func TestLoxfmt(t *testing.T) {
	loxfmtPath := loxtest.MustBuildBinary(t, "loxfmt")
	runner := newRunner(loxfmtPath)
	loxtest.Run(t, runner)
}

func newRunner(loxfmtPath string) *runner {
	return &runner{
		loxfmtPath: loxfmtPath,
	}
}

type runner struct {
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
	t.Logf("go run ./loxfmt %s", strings.Join(args, " "))

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
