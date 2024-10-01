package test

import (
	"bytes"
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
		t.Skip("formatter not specified with the -formatter flag")
	}
	runTests(t, formatterRunner{}, "testdata")
}

type formatterRunner struct{}

func (r formatterRunner) Test(t *testing.T, path string) {
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
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
	if err != nil {
		t.Fatal(err)
	}

	return stdout
}

func (r formatterRunner) Update(t *testing.T, path string) {
	r.runFormatter(t, path, "-w")
}
