package test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

var formatter = flag.String("formatter", "", "path to the formatter to test")

func TestFormatter(t *testing.T) {
	if *formatter == "" {
		t.Skip("-formatter flag not provided")
	}
	runTests(t, newFormatterRunner(*pwd, *formatter), "testdata")
}

func newFormatterRunner(pwd string, formatter string) *formatterRunner {
	return &formatterRunner{
		pwd:       pwd,
		formatter: formatter,
	}
}

type formatterRunner struct {
	pwd       string
	formatter string
}

func (r *formatterRunner) Test(t *testing.T, path string) {
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(want, []byte(syntaxErrorComment)) {
		t.Skipf("file is marked with %s", syntaxErrorComment)
	}

	got := r.runFormatter(t, path)

	if !bytes.Equal(want, got) {
		t.Errorf("incorrect output printed to stdout:\n%s", computeTextDiff(string(want), string(got)))
	}
}

func (r *formatterRunner) runFormatter(t *testing.T, path string, flags ...string) []byte {
	args := append(slices.Clone(flags), path)
	cmd := exec.Command(*formatter, args...)
	relFormatter, err := filepath.Rel(r.pwd, r.formatter)
	if err != nil {
		t.Fatal(err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	relPath, err := filepath.Rel(r.pwd, absPath)
	if err != nil {
		t.Fatal(err)
	}
	argsWithRelPath := append(slices.Clone(flags), relPath)
	t.Logf("%s %s", relFormatter, strings.Join(argsWithRelPath, " "))
	t.Logf("go run ./loxfmt %s", strings.Join(argsWithRelPath, " "))

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

func (r *formatterRunner) Update(t *testing.T, path string) {
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(contents, []byte(syntaxErrorComment)) {
		t.Skipf("file is marked with %s", syntaxErrorComment)
	}
	r.runFormatter(t, path, "-write")
}
