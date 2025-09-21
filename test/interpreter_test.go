package test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

var (
	interpreter = flag.String("interpreter", "", "path to the interpreter to test")
	hints       = flag.Bool("hints", false, "test the interpreter's hints")

	printsRe = regexp.MustCompile(`// prints: (.+)`)
	errorRe  = regexp.MustCompile(`// error: (.+)`)
	hintRe   = regexp.MustCompile(`// hint: (.+)`)
)

func TestInterpreter(t *testing.T) {
	if *interpreter == "" {
		t.Skip("-interpreter flag not provided")
	}
	runTests(t, newInterpreterRunner(*pwd, *interpreter), "testdata")
}

func TestInterpreterHints(t *testing.T) {
	if *interpreter == "" {
		t.Skip("-interpreter flag not provided")
	}
	if !*hints {
		t.Skip("-hints flag not provided")
	}
	runTests(t, newInterpreterHintsRunner(*pwd, *interpreter), "testdata")
}

func newInterpreterRunner(pwd string, interpreter string) *interpreterRunner {
	return &interpreterRunner{
		pwd:         pwd,
		interpreter: interpreter,
	}
}

func newInterpreterHintsRunner(pwd string, interpreter string) *interpreterRunner {
	return &interpreterRunner{
		pwd:         pwd,
		interpreter: interpreter,
		hints:       true,
	}
}

type interpreterRunner struct {
	pwd         string
	interpreter string
	hints       bool
}

func (r *interpreterRunner) Test(t *testing.T, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(data, []byte(syntaxErrorComment)) {
		t.Skipf("file is marked with %s", syntaxErrorComment)
	}

	want := r.parseExpectedResult(data)
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

	if !cmp.Equal(want.Hints, got.Hints) {
		t.Errorf("incorrect hints printed to stderr:\n%s", computeDiff(want.Hints, got.Hints))
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

func (r *interpreterRunner) runInterpreter(t *testing.T, path string) *interpreterResult {
	var args []string
	if r.hints {
		args = append(args, "-hints")
	}
	args = append(args, path)
	cmd := exec.Command(r.interpreter, args...)
	relInterpeter, err := filepath.Rel(r.pwd, r.interpreter)
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
	flags := ""
	if len(args) > 1 {
		flags = strings.Join(args[:len(args)-1], " ") + " "
	}
	t.Logf("%s %s%s", relInterpeter, flags, relPath)
	t.Logf("go run ./golox %s%s", flags, relPath)

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

func (r *interpreterRunner) parseExpectedResult(data []byte) *interpreterResult {
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

func (r *interpreterRunner) parseExpectedStdout(data []byte) []byte {
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

func (r *interpreterRunner) parseExpectedErrors(data []byte) [][]byte {
	if r.hints {
		return nil
	}
	var errors [][]byte
	for _, match := range errorRe.FindAllSubmatch(data, -1) {
		errors = append(errors, match[1])
	}
	return errors
}

func (r *interpreterRunner) parseExpectedHints(data []byte) [][]byte {
	if !r.hints {
		return nil
	}
	var hints [][]byte
	for _, match := range hintRe.FindAllSubmatch(data, -1) {
		hints = append(hints, match[1])
	}
	return hints
}

func (r *interpreterRunner) Update(t *testing.T, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.HasPrefix(data, []byte(syntaxErrorComment)) {
		t.Skipf("file is marked with %s", syntaxErrorComment)
	}

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
		if len(result.Hints) > 0 {
			t.Logf("hints:\n%s", bytes.Join(result.Hints, []byte("\n")))
		} else {
			t.Logf("hints: <empty>")
		}
	} else {
		t.Logf("stderr: <empty>")
	}

	if r.hints {
		data = r.updateExpectedHints(t, path, data, result.Hints)
	} else {
		data = r.updateExpectedStdout(t, path, data, result.Stdout)
		data = r.updateExpectedErrors(t, path, data, result.Errors)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func (r *interpreterRunner) updateExpectedStdout(t *testing.T, path string, data []byte, stdout []byte) []byte {
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

func (r *interpreterRunner) updateExpectedErrors(t *testing.T, path string, data []byte, errors [][]byte) []byte {
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

func (r *interpreterRunner) updateExpectedHints(t *testing.T, path string, data []byte, hints [][]byte) []byte {
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
