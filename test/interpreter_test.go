package test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	interpreter = flag.String("interpreter", "", "path to the interpreter to test")

	printsRe = regexp.MustCompile(`// prints: (.+)`)
	errorRe  = regexp.MustCompile(`// error: (.+)`)
)

func TestInterpreter(t *testing.T) {
	if *interpreter == "" {
		t.Skip("interpreter not specified with the -interpreter flag")
	}
	runTests(t, interpreterRunner{}, "testdata")
}

type interpreterRunner struct{}

func (r interpreterRunner) Test(t *testing.T, path string) {
	want := r.parseExpectedResult(t, path)
	got := r.runInterpreter(t, path)

	if want.ExitCode != got.ExitCode {
		t.Errorf("exit code = %d, want %d", got.ExitCode, want.ExitCode)
		t.Logf("stdout:\n%s", got.Stdout)
		t.Logf("stderr:\n%s", got.Stderr)
		return
	}

	if !bytes.Equal(want.Stdout, got.Stdout) {
		t.Errorf("incorrect output printed to stdout:\n%s", computeTextDiff(string(want.Stdout), string(got.Stdout)))
	}

	if !cmp.Equal(want.Errors, got.Errors) {
		t.Errorf("incorrect errors printed to stderr:\n%s", computeDiff(want.Errors, got.Errors))
		t.Errorf("stderr:\n%s", got.Stderr)
	}
}

type interpreterResult struct {
	Stdout   []byte
	Stderr   []byte
	Errors   [][]byte
	ExitCode int
}

func (r interpreterRunner) runInterpreter(t *testing.T, path string) interpreterResult {
	cmd := exec.Command(*interpreter, path)
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s %s", *interpreter, absPath)

	stdout, err := cmd.Output()

	exitErr := &exec.ExitError{}
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	var errors [][]byte
	errorRe := regexp.MustCompile(`(?m)^.+:\d+:\d+: error: (.+)$`)
	for _, match := range errorRe.FindAllSubmatch(exitErr.Stderr, -1) {
		errors = append(errors, match[1])
	}

	return interpreterResult{
		Stdout:   stdout,
		Stderr:   exitErr.Stderr,
		Errors:   errors,
		ExitCode: cmd.ProcessState.ExitCode(),
	}
}

func (r interpreterRunner) parseExpectedResult(t *testing.T, path string) interpreterResult {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	errors := r.parseExpectedErrors(data)

	result := interpreterResult{
		Stdout: r.parseExpectedStdout(data),
		Errors: errors,
	}
	if len(result.Errors) > 0 {
		result.ExitCode = 1
	}

	return result
}

func (r interpreterRunner) parseExpectedStdout(data []byte) []byte {
	var b bytes.Buffer
	for _, match := range printsRe.FindAllSubmatch(data, -1) {
		if !bytes.Equal(match[1], []byte("<empty>")) {
			b.Write(match[1])
		}
		b.WriteRune('\n')
	}
	return b.Bytes()
}

func (r interpreterRunner) parseExpectedErrors(data []byte) [][]byte {
	var errors [][]byte
	for _, match := range errorRe.FindAllSubmatch(data, -1) {
		errors = append(errors, match[1])
	}
	return errors
}

func (r interpreterRunner) Update(t *testing.T, path string) {
	t.Logf("updating expected output for %s", path)

	result := r.runInterpreter(t, path)

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

	data = r.updateExpectedStdout(t, path, data, result.Stdout)
	data = r.updateExpectedErrors(t, path, data, result.Errors)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func (r interpreterRunner) updateExpectedStdout(t *testing.T, path string, data []byte, stdout []byte) []byte {
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

func (r interpreterRunner) updateExpectedErrors(t *testing.T, path string, data []byte, errors [][]byte) []byte {
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
