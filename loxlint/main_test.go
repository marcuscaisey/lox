package main_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"

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

	if want.ExitCode != got.ExitCode {
		t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", got.ExitCode, want.ExitCode, got.Stdout, got.Stderr)
	}

	if len(got.Stdout) > 0 {
		t.Errorf("no output expected to be printed to stdout\nstdout:\n%s", got.Stdout)
	}

	if !cmp.Equal(want.Hints, got.Hints) {
		t.Errorf("incorrect hints printed to stderr:\n%s\nstderr:\n%s", loxtest.ComputeDiff(want.Hints, got.Hints), got.Stderr)
	}
}

type loxlintResult struct {
	Stdout   []byte
	Stderr   []byte
	Errors   [][]byte
	Warnings [][]byte
	Hints    [][]byte
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
	var errors [][]byte
	var warnings [][]byte
	var hints [][]byte
	errorWarningHintRe := regexp.MustCompile(`(?m)^\d+:\d+: (error|warning|hint): (.+)$`)
	for _, match := range errorWarningHintRe.FindAllSubmatch(exitErr.Stderr, -1) {
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
		Stdout:   stdout,
		Stderr:   exitErr.Stderr,
		Errors:   errors,
		Warnings: warnings,
		Hints:    hints,
		ExitCode: cmd.ProcessState.ExitCode(),
	}
}

func (r *runner) mustParseExpectedResult(t *testing.T, path string) *loxlintResult {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	result := &loxlintResult{
		Errors:   r.parseExpectedErrors(data),
		Warnings: r.parseExpectedWarnings(data),
		Hints:    r.parseExpectedHints(data),
	}
	if len(result.Errors)+len(result.Warnings)+len(result.Hints) > 0 {
		result.ExitCode = 1
	}

	return result
}

func (r *runner) parseExpectedErrors(data []byte) [][]byte {
	var errors [][]byte
	for _, match := range errorRe.FindAllSubmatch(data, -1) {
		errors = append(errors, match[1])
	}
	return errors
}

func (r *runner) parseExpectedWarnings(data []byte) [][]byte {
	var warnings [][]byte
	for _, match := range warningRe.FindAllSubmatch(data, -1) {
		warnings = append(warnings, match[1])
	}
	return warnings
}

func (r *runner) parseExpectedHints(data []byte) [][]byte {
	var hints [][]byte
	for _, match := range hintRe.FindAllSubmatch(data, -1) {
		hints = append(hints, match[1])
	}
	return hints
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
		if len(result.Warnings) > 0 {
			t.Logf("warnings:\n%s", bytes.Join(result.Warnings, []byte("\n")))
		} else {
			t.Logf("warnings: <empty>")
		}
		if len(result.Hints) > 0 {
			t.Logf("hints:\n%s", bytes.Join(result.Hints, []byte("\n")))
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
