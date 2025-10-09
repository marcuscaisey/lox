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
	hintRe = regexp.MustCompile(`// hint: (.+)`)
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
	var hints [][]byte
	hintRe := regexp.MustCompile(`(?m)^\d+:\d+: hint: (.+)$`)
	for _, match := range hintRe.FindAllSubmatch(exitErr.Stderr, -1) {
		hints = append(hints, match[1])
	}

	return &loxlintResult{
		Stdout:   stdout,
		Stderr:   exitErr.Stderr,
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
		Stdout: nil,
		Hints:  r.parseExpectedHints(data),
	}
	if len(result.Hints) > 0 {
		result.ExitCode = 1
	}

	return result
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

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = r.mustUpdateExpectedHints(t, path, data, result.Hints)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func (r *runner) mustUpdateExpectedHints(t *testing.T, path string, data []byte, hints [][]byte) []byte {
	matches := hintRe.FindAllSubmatchIndex(data, -1)
	if len(hints) != len(matches) {
		t.Fatalf(`%d "// hint:" %s found in %s but %d %s printed to stderr, these should be equal`,
			len(matches), pluralise("comment", len(matches)), path, len(hints), pluralise("hint", len(hints)))
	}
	if len(hints) == 0 {
		return data
	}

	var b bytes.Buffer
	lastEnd := 0
	for i, match := range matches {
		start, end := match[2], match[3]
		b.Write(data[lastEnd:start])
		b.Write(hints[i])
		lastEnd = end
	}
	b.Write(data[lastEnd:])

	return b.Bytes()
}

func pluralise(s string, n int) string {
	if n == 1 {
		return s
	}
	return s + "s"
}
