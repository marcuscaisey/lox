package test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var (
	noFormatComment = "// noformat"
)

type formatterRunner struct{}

func (r formatterRunner) Test(t *testing.T, path string) {
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(want, []byte(noFormatComment)) {
		t.Skipf("file is marked with %s", noFormatComment)
	}

	got := r.runFormatter(t, path)

	if !bytes.Equal(want, got) {
		t.Errorf("incorrect output printed to stdout:\n%s", computeTextDiff(string(want), string(got)))
	}
}

func (r formatterRunner) runFormatter(t *testing.T, path string, flags ...string) []byte {
	args := append(slices.Clone(flags), path)
	cmd := exec.Command(*formatter, args...)
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	argsWithAbsPath := append(slices.Clone(flags), absPath)
	t.Logf("%s %s", *formatter, strings.Join(argsWithAbsPath, " "))

	stdout, err := cmd.Output()
	exitErr := &exec.ExitError{}
	if errors.As(err, &exitErr) {
		t.Fatalf("%s\n%s", err, exitErr.Stderr)
	}
	if err != nil {
		t.Fatal(err)
	}

	return stdout
}

func (r formatterRunner) Update(t *testing.T, path string) {
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(contents, []byte(noFormatComment)) {
		t.Skipf("file is marked with %s", noFormatComment)
	}
	r.runFormatter(t, path, "-w")
}
