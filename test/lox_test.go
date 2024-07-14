package test

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/google/go-cmp/cmp"
)

var (
	interpreter = flag.String("interpreter", "", "path to the interpreter to test")
	update      = flag.Bool("update", false, "updates the expected output of each test")

	printsRe = regexp.MustCompile(`// prints: (.+)`)
	errorRe  = regexp.MustCompile(`// error: (.+)`)
)

func TestMain(m *testing.M) {
	flag.Parse()
	if *interpreter == "" {
		fmt.Fprintln(os.Stderr, "interpreter must be specified with the -interpreter flag")
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestLox(t *testing.T) {
	runTests(t, "testdata")
}

func runTests(t *testing.T, path string) {
	matches, err := filepath.Glob(filepath.Join(path, "*"))
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range matches {
		path := path
		testName := snakeToPascalCase(filepath.Base(path))
		if filepath.Ext(path) == ".lox" {
			testName = strings.TrimSuffix(testName, ".lox")
			t.Run(testName, func(t *testing.T) {
				t.Parallel()
				if *update {
					updateExpectedOutput(t, path)
				} else {
					runTest(t, path)
				}
			})
		} else {
			t.Run(testName, func(t *testing.T) {
				t.Parallel()
				runTests(t, path)
			})
		}

	}
}

func snakeToPascalCase(s string) string {
	var b strings.Builder
	for _, part := range strings.Split(s, "_") {
		r, size := utf8.DecodeRuneInString(part)
		b.WriteRune(unicode.ToUpper(r))
		b.WriteString(part[size:])
	}
	return b.String()
}

type result struct {
	Stdout   []byte
	Stderr   []byte
	Errors   [][]byte
	ExitCode int
}

func runTest(t *testing.T, path string) {
	want := parseExpectedResult(t, path)
	got := runInterpreter(t, path)

	if want.ExitCode != got.ExitCode {
		t.Errorf("exit code = %d, want %d", got.ExitCode, want.ExitCode)
		t.Logf("stdout:\n%s", got.Stdout)
		t.Logf("stderr:\n%s", got.Stderr)
		return
	}

	if !bytes.Equal(want.Stdout, got.Stdout) {
		t.Errorf("incorrect output printed to stdout:\n%s", computeDiff(want.Stdout, got.Stdout))
	}

	if !cmp.Equal(want.Errors, got.Errors) {
		t.Errorf("incorrect errors printed to stderr:\n%s", computeDiff(want.Errors, got.Errors))
		t.Errorf("stderr:\n%s", got.Stderr)
	}
}

func runInterpreter(t *testing.T, path string) result {
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

	return result{
		Stdout:   stdout,
		Stderr:   exitErr.Stderr,
		Errors:   errors,
		ExitCode: cmd.ProcessState.ExitCode(),
	}
}

func computeDiff(want, got any) string {
	color.NoColor = false
	diff := cmp.Diff(want, got, cmp.Transformer("BytesToString", func(b []byte) string {
		return string(b)
	}))
	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "-") {
			lines[i] = color.GreenString(line)
		} else if strings.HasPrefix(line, "+") {
			lines[i] = color.RedString(line)
		}
	}
	diff = strings.Join(lines, "\n")
	return fmt.Sprint(color.GreenString("want -\n"), color.RedString("got +\n"), diff)
}

func parseExpectedResult(t *testing.T, path string) result {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	errors := parseExpectedErrors(data)

	r := result{
		Stdout: parseExpectedStdout(data),
		Errors: errors,
	}
	if len(r.Errors) > 0 {
		r.ExitCode = 1
	}

	return r
}

func parseExpectedStdout(data []byte) []byte {
	var b bytes.Buffer
	for _, match := range printsRe.FindAllSubmatch(data, -1) {
		if !bytes.Equal(match[1], []byte("<empty>")) {
			b.Write(match[1])
		}
		b.WriteRune('\n')
	}
	return b.Bytes()
}

func parseExpectedErrors(data []byte) [][]byte {
	var errors [][]byte
	for _, match := range errorRe.FindAllSubmatch(data, -1) {
		errors = append(errors, match[1])
	}
	return errors
}

func updateExpectedOutput(t *testing.T, path string) {
	t.Logf("updating expected output for %s", path)

	result := runInterpreter(t, path)

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

	data = updateExpectedStdout(t, path, data, result.Stdout)
	data = updateExpectedErrors(t, path, data, result.Errors)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func updateExpectedStdout(t *testing.T, path string, data []byte, stdout []byte) []byte {
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

func updateExpectedErrors(t *testing.T, path string, data []byte, errors [][]byte) []byte {
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
