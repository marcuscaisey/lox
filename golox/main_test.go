package main_test

import (
	"errors"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/marcuscaisey/lox/test/loxtest"
)

var (
	printsRe = regexp.MustCompile(`// prints: (.+)`)
	errorRe  = regexp.MustCompile(`// error: (.+)`)
)

func TestGolox(t *testing.T) {
	goloxPath := loxtest.MustBuildBinary(t, "golox")
	runner := newRunner(goloxPath)
	loxtest.Run(t, runner, loxtest.WithSkipSyntaxErrors(false))
}

func newRunner(goloxPath string) *runner {
	return &runner{
		goloxPath: goloxPath,
	}
}

type runner struct {
	goloxPath string
}

func (r *runner) Test(t *testing.T, path string) {
	want := r.mustParseExpectedResult(t, path)
	got := r.mustRunGolox(t, path)

	if got.ExitCode != want.ExitCode {
		t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", got.ExitCode, want.ExitCode, got.Stdout, got.Stderr)
	}

	if diff := loxtest.TextDiff(got.Stdout, want.Stdout); diff != "" {
		t.Errorf("incorrect output printed to stdout:\n%s\nstdout:\n%s", diff, got.Stdout)
	}

	if diff := loxtest.LinesDiff(got.Errors, want.Errors); diff != "" {
		t.Errorf("incorrect errors printed to stderr:\n%s\nstderr:\n%s", diff, got.Stderr)
	}
}

type goloxResult struct {
	Stdout   string
	Stderr   []byte
	Errors   []string
	ExitCode int
}

func (r *runner) mustRunGolox(t *testing.T, path string) *goloxResult {
	cmd := exec.Command(r.goloxPath, path)
	t.Logf("go run ./golox %s", path)

	stdout, err := cmd.Output()

	exitErr := &exec.ExitError{}
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	var errors []string
	errorRe := regexp.MustCompile(`(?m)^\d+:\d+: error: (.+)$`)
	for _, match := range errorRe.FindAllStringSubmatch(string(exitErr.Stderr), -1) {
		errors = append(errors, match[1])
	}

	return &goloxResult{
		Stdout:   string(stdout),
		Stderr:   exitErr.Stderr,
		Errors:   errors,
		ExitCode: cmd.ProcessState.ExitCode(),
	}
}

func (r *runner) mustParseExpectedResult(t *testing.T, path string) *goloxResult {
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	stdoutLines := loxtest.ParseComments(contents, printsRe)
	stdout := strings.Join(stdoutLines, "\n")
	if stdout != "" {
		stdout += "\n"
	}
	result := &goloxResult{
		Stdout: stdout,
		Errors: loxtest.ParseComments(contents, errorRe),
	}
	if len(result.Errors) > 0 {
		result.ExitCode = 1
	}

	return result
}

func (r *runner) Update(t *testing.T, path string) {
	t.Logf("updating expected output for %s", path)

	result := r.mustRunGolox(t, path)

	t.Logf("exit code: %d", result.ExitCode)
	if len(result.Stdout) > 0 {
		t.Logf("stdout:\n%s", result.Stdout)
	} else {
		t.Logf("stdout: <empty>")
	}
	if len(result.Stderr) > 0 {
		t.Logf("stderr:\n%s", result.Stderr)
		if len(result.Errors) > 0 {
			t.Logf("errors:\n%s", strings.Join(result.Errors, "\n"))
		} else {
			t.Logf("errors: <empty>")
		}
	} else {
		t.Logf("stderr: <empty>")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout := strings.TrimSuffix(result.Stdout, "\n")
	var stdoutLines []string
	if stdout != "" {
		stdoutLines = strings.Split(stdout, "\n")
	}
	contents = loxtest.MustUpdateComments(t, path, contents, printsRe, stdoutLines)
	contents = loxtest.MustUpdateComments(t, path, contents, errorRe, result.Errors)

	if err := os.WriteFile(path, contents, 0644); err != nil {
		t.Fatal(err)
	}
}
