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
	errorRe   = regexp.MustCompile(`// lint error: (.+)`)
	warningRe = regexp.MustCompile(`// lint warning: (.+)`)
	hintRe    = regexp.MustCompile(`// lint hint: (.+)`)
)

func TestLoxlint(t *testing.T) {
	loxlintPath := loxtest.MustBuildBinary(t, "loxlint")
	runner := newRunner(loxlintPath)
	loxtest.Run(t, runner)
}

func newRunner(loxlintPath string) *runner {
	return &runner{
		loxlintPath: loxlintPath,
	}
}

type runner struct {
	loxlintPath string
}

func (r *runner) Test(t *testing.T, path string) {
	want := r.mustParseExpectedResult(t, path)
	got := r.mustRunLoxlint(t, path)

	if got.ExitCode != want.ExitCode {
		t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", got.ExitCode, want.ExitCode, got.Stdout, got.Stderr)
	}

	if diff := loxtest.TextDiff(got.Stdout, want.Stdout); diff != "" {
		t.Errorf("incorrect output printed to stdout:\n%s\nstdout:\n%s", diff, got.Stdout)
	}

	printStderr := false
	if diff := loxtest.LinesDiff(got.Errors, want.Errors); diff != "" {
		printStderr = true
		t.Errorf("incorrect errors printed to stderr:\n%s", diff)
	}

	if diff := loxtest.LinesDiff(got.Warnings, want.Warnings); diff != "" {
		printStderr = true
		t.Errorf("incorrect warnings printed to stderr:\n%s", diff)
	}

	if diff := loxtest.LinesDiff(got.Hints, want.Hints); diff != "" {
		printStderr = true
		t.Errorf("incorrect hints printed to stderr:\n%s", diff)
	}

	if printStderr {
		t.Logf("stderr:\n%s", got.Stderr)
	}
}

type loxlintResult struct {
	Stdout   string
	Stderr   []byte
	Errors   []string
	Warnings []string
	Hints    []string
	ExitCode int
}

func (r *runner) mustRunLoxlint(t *testing.T, path string) *loxlintResult {
	cmd := exec.Command(r.loxlintPath, path)
	t.Logf("go run ./loxlint %s", path)

	stdout, err := cmd.Output()

	exitErr := &exec.ExitError{}
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	var errors []string
	var warnings []string
	var hints []string
	errorWarningHintRe := regexp.MustCompile(`(?m)^\d+:\d+: (error|warning|hint): (.+)$`)
	for _, match := range errorWarningHintRe.FindAllStringSubmatch(string(exitErr.Stderr), -1) {
		switch string(match[1]) {
		case "error":
			errors = append(errors, match[2])
		case "warning":
			warnings = append(warnings, match[2])
		case "hint":
			hints = append(hints, match[2])
		}
	}

	return &loxlintResult{
		Stdout:   string(stdout),
		Stderr:   exitErr.Stderr,
		Errors:   errors,
		Warnings: warnings,
		Hints:    hints,
		ExitCode: cmd.ProcessState.ExitCode(),
	}
}

func (r *runner) mustParseExpectedResult(t *testing.T, path string) *loxlintResult {
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	result := &loxlintResult{
		Errors:   loxtest.ParseComments(contents, errorRe),
		Warnings: loxtest.ParseComments(contents, warningRe),
		Hints:    loxtest.ParseComments(contents, hintRe),
	}
	if len(result.Errors)+len(result.Warnings)+len(result.Hints) > 0 {
		result.ExitCode = 1
	}

	return result
}

func (r *runner) Update(t *testing.T, path string) {
	t.Logf("updating expected output for %s", path)

	result := r.mustRunLoxlint(t, path)

	t.Logf("exit code: %d", result.ExitCode)
	if len(result.Stdout) > 0 {
		t.Errorf("no output expected to be printed to stdout\nstdout:\n%s", result.Stdout)
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
		if len(result.Warnings) > 0 {
			t.Logf("warnings:\n%s", strings.Join(result.Warnings, "\n"))
		} else {
			t.Logf("warnings: <empty>")
		}
		if len(result.Hints) > 0 {
			t.Logf("hints:\n%s", strings.Join(result.Hints, "\n"))
		} else {
			t.Logf("hints: <empty>")
		}
	} else {
		t.Logf("stderr: <empty>")
	}
	if t.Failed() {
		return
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	contents = loxtest.MustUpdateComments(t, path, contents, errorRe, result.Errors)
	contents = loxtest.MustUpdateComments(t, path, contents, warningRe, result.Warnings)
	contents = loxtest.MustUpdateComments(t, path, contents, hintRe, result.Hints)

	if err := os.WriteFile(path, contents, 0644); err != nil {
		t.Fatal(err)
	}
}
