package main_test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/marcuscaisey/lox/test/loxtest"
)

var (
	testHints = flag.Bool("test-hints", false, "test the interpreter's hints")

	printsRe = regexp.MustCompile(`// prints: (.+)`)
	errorRe  = regexp.MustCompile(`// error: (.+)`)
	hintRe   = regexp.MustCompile(`// hint: (.+)`)
)

func TestGolox(t *testing.T) {
	goloxPath := loxtest.MustBuildBinary(t, "golox")
	if *testHints {
		t.Skip("-test-hints flag provided")
	}
	runner := newRunner(goloxPath)
	loxtest.Run(t, runner, loxtest.WithSkipSyntaxErrors(false))
}

func TestGoloxHints(t *testing.T) {
	path := loxtest.MustBuildBinary(t, "golox")
	if !*testHints {
		t.Skip("-test-hints flag not provided")
	}
	runner := newHintsRunner(path)
	loxtest.Run(t, runner)
}

func newRunner(goloxPath string) *runner {
	return &runner{
		goloxPath: goloxPath,
	}
}

func newHintsRunner(goloxPath string) *runner {
	return &runner{
		goloxPath: goloxPath,
		hints:     true,
	}
}

type runner struct {
	goloxPath string
	hints     bool
}

func (r *runner) Test(t *testing.T, path string) {
	want := r.mustParseExpectedResult(t, path)
	got := r.mustRunInterpreter(t, path)

	if want.ExitCode != got.ExitCode {
		t.Errorf("exit code = %d, want %d", got.ExitCode, want.ExitCode)
		t.Logf("stdout:\n%s", got.Stdout)
		t.Logf("stderr:\n%s", got.Stderr)
		return
	}

	if !bytes.Equal(want.Stdout, got.Stdout) {
		t.Errorf("incorrect output printed to stdout:\n%s", loxtest.ComputeTextDiff(string(want.Stdout), string(got.Stdout)))
	}

	if !cmp.Equal(want.Errors, got.Errors) {
		t.Errorf("incorrect errors printed to stderr:\n%s", loxtest.ComputeDiff(want.Errors, got.Errors))
		t.Errorf("stderr:\n%s", got.Stderr)
	}

	if !cmp.Equal(want.Hints, got.Hints) {
		t.Errorf("incorrect hints printed to stderr:\n%s", loxtest.ComputeDiff(want.Hints, got.Hints))
		t.Errorf("stderr:\n%s", got.Stderr)
	}
}

type interpreterResult struct {
	Stdout   []byte
	Stderr   []byte
	Errors   [][]byte
	Hints    [][]byte
	ExitCode int
}

func (r *runner) mustRunInterpreter(t *testing.T, path string) *interpreterResult {
	var args []string
	if r.hints {
		args = append(args, "-hints")
	}
	args = append(args, path)
	cmd := exec.Command(r.goloxPath, args...)
	t.Logf("go run ./golox %s", strings.Join(args, " "))

	stdout, err := cmd.Output()

	exitErr := &exec.ExitError{}
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	var errors [][]byte
	var hints [][]byte
	errorHintRe := regexp.MustCompile(`(?m)^\d+:\d+: (error|hint): (.+)$`)
	for _, match := range errorHintRe.FindAllSubmatch(exitErr.Stderr, -1) {
		switch string(match[1]) {
		case "hint":
			hints = append(hints, match[2])
		case "error":
			errors = append(errors, match[2])
		}
	}

	return &interpreterResult{
		Stdout:   stdout,
		Stderr:   exitErr.Stderr,
		Errors:   errors,
		Hints:    hints,
		ExitCode: cmd.ProcessState.ExitCode(),
	}
}

func (r *runner) mustParseExpectedResult(t *testing.T, path string) *interpreterResult {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	result := &interpreterResult{
		Stdout: r.parseExpectedStdout(data),
		Errors: r.parseExpectedErrors(data),
		Hints:  r.parseExpectedHints(data),
	}
	if len(result.Errors)+len(result.Hints) > 0 {
		result.ExitCode = 1
	}

	return result
}

func (r *runner) parseExpectedStdout(data []byte) []byte {
	if r.hints {
		return nil
	}
	var b bytes.Buffer
	for _, match := range printsRe.FindAllSubmatch(data, -1) {
		if !bytes.Equal(match[1], []byte("<empty>")) {
			b.Write(match[1])
		}
		b.WriteRune('\n')
	}
	return b.Bytes()
}

func (r *runner) parseExpectedErrors(data []byte) [][]byte {
	if r.hints {
		return nil
	}
	var errors [][]byte
	for _, match := range errorRe.FindAllSubmatch(data, -1) {
		errors = append(errors, match[1])
	}
	return errors
}

func (r *runner) parseExpectedHints(data []byte) [][]byte {
	if !r.hints {
		return nil
	}
	var hints [][]byte
	for _, match := range hintRe.FindAllSubmatch(data, -1) {
		hints = append(hints, match[1])
	}
	return hints
}

func (r *runner) Update(t *testing.T, path string) {
	t.Logf("updating expected output for %s", path)

	result := r.mustRunInterpreter(t, path)

	t.Logf("exit code: %d", result.ExitCode)
	if len(result.Stdout) > 0 {
		t.Logf("stdout:\n%s", result.Stdout)
	} else {
		t.Logf("stdout: <empty>")
	}
	if len(result.Stderr) > 0 {
		t.Logf("stderr:\n%s", result.Stderr)
		if len(result.Errors) > 0 {
			t.Logf("errors:\n%s", bytes.Join(result.Errors, []byte("\n")))
		} else {
			t.Logf("errors: <empty>")
		}
		if len(result.Hints) > 0 {
			t.Logf("hints:\n%s", bytes.Join(result.Hints, []byte("\n")))
		} else {
			t.Logf("hints: <empty>")
		}
	} else {
		t.Logf("stderr: <empty>")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if r.hints {
		data = r.mustUpdateExpectedHints(t, path, data, result.Hints)
	} else {
		data = r.mustUpdateExpectedStdout(t, path, data, result.Stdout)
		data = r.mustUpdateExpectedErrors(t, path, data, result.Errors)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func (r *runner) mustUpdateExpectedStdout(t *testing.T, path string, data []byte, stdout []byte) []byte {
	var lines [][]byte
	if len(stdout) > 0 {
		lines = bytes.Split(bytes.TrimSuffix(stdout, []byte("\n")), []byte("\n"))
	}
	matches := printsRe.FindAllSubmatchIndex(data, -1)
	if len(lines) != len(matches) {
		t.Fatalf(`%d "// prints" %s found in %s but %d %s printed to stdout, these should be equal`,
			len(matches), pluralise("comment", len(matches)), path, len(lines), pluralise("line", len(lines)))
	}
	if len(stdout) == 0 {
		return data
	}

	var b bytes.Buffer
	lastEnd := 0
	for i, match := range matches {
		start, end := match[2], match[3]
		b.Write(data[lastEnd:start])
		if bytes.Equal(lines[i], []byte("")) {
			b.WriteString("<empty>")
		} else {
			b.Write(lines[i])
		}
		lastEnd = end
	}
	b.Write(data[lastEnd:])

	return b.Bytes()
}

func (r *runner) mustUpdateExpectedErrors(t *testing.T, path string, data []byte, errors [][]byte) []byte {
	matches := errorRe.FindAllSubmatchIndex(data, -1)
	if len(errors) != len(matches) {
		t.Fatalf(`%d "// error:" %s found in %s but %d %s printed to stderr, these should be equal`,
			len(matches), pluralise("comment", len(matches)), path, len(errors), pluralise("error", len(errors)))
	}
	if len(errors) == 0 {
		return data
	}

	var b bytes.Buffer
	lastEnd := 0
	for i, match := range matches {
		start, end := match[2], match[3]
		b.Write(data[lastEnd:start])
		b.Write(errors[i])
		lastEnd = end
	}
	b.Write(data[lastEnd:])

	return b.Bytes()
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
