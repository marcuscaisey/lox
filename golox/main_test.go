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

	if want.ExitCode != got.ExitCode {
		t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", got.ExitCode, want.ExitCode, got.Stdout, got.Stderr)
	}

	if !bytes.Equal(want.Stdout, got.Stdout) {
		t.Errorf("incorrect output printed to stdout:\n%s", loxtest.ComputeTextDiff(string(want.Stdout), string(got.Stdout)))
	}

	if !cmp.Equal(want.Errors, got.Errors) {
		t.Errorf("incorrect errors printed to stderr:\n%s\nstderr:\n%s", loxtest.ComputeDiff(want.Errors, got.Errors), got.Stderr)
	}
}

type goloxResult struct {
	Stdout   []byte
	Stderr   []byte
	Errors   [][]byte
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
	var errors [][]byte
	errorRe := regexp.MustCompile(`(?m)^\d+:\d+: error: (.+)$`)
	for _, match := range errorRe.FindAllSubmatch(exitErr.Stderr, -1) {
		errors = append(errors, match[1])
	}

	return &goloxResult{
		Stdout:   stdout,
		Stderr:   exitErr.Stderr,
		Errors:   errors,
		ExitCode: cmd.ProcessState.ExitCode(),
	}
}

func (r *runner) mustParseExpectedResult(t *testing.T, path string) *goloxResult {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	result := &goloxResult{
		Stdout: r.parseExpectedStdout(data),
		Errors: r.parseExpectedErrors(data),
	}
	if len(result.Errors) > 0 {
		result.ExitCode = 1
	}

	return result
}

func (r *runner) parseExpectedStdout(data []byte) []byte {
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
	var errors [][]byte
	for _, match := range errorRe.FindAllSubmatch(data, -1) {
		errors = append(errors, match[1])
	}
	return errors
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
			t.Logf("errors:\n%s", bytes.Join(result.Errors, []byte("\n")))
		} else {
			t.Logf("errors: <empty>")
		}
	} else {
		t.Logf("stderr: <empty>")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = r.mustUpdateExpectedStdout(t, path, data, result.Stdout)
	data = r.mustUpdateExpectedErrors(t, path, data, result.Errors)

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

func pluralise(s string, n int) string {
	if n == 1 {
		return s
	}
	return s + "s"
}
