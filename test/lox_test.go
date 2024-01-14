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
	"github.com/sergi/go-diff/diffmatchpatch"
)

var interpreter = flag.String("interpreter", "", "path to the interpreter to test")
var update = flag.Bool("update", false, "updates the expected output of each test")

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
	Stdout, Stderr []byte
	ExitCode       int
}

func runTest(t *testing.T, path string) {
	want := parseExpectedResult(t, path)
	got := runInterpreter(t, path)

	if want.ExitCode != got.ExitCode {
		t.Errorf("incorrect exit code: want %d, got %d", want.ExitCode, got.ExitCode)
	}

	if !bytes.Equal(want.Stdout, got.Stdout) {
		t.Errorf("incorrect output printed to stdout:\n%s", computeDiff(want.Stdout, got.Stdout))
	}

	if !bytes.Equal(want.Stderr, got.Stderr) {
		t.Errorf("incorrect output printed to stderr:\n%s", computeDiff(want.Stderr, got.Stderr))
	}
}

func runInterpreter(t *testing.T, path string) result {
	cmd := exec.Command(*interpreter, path)
	t.Logf("running %s %s", *interpreter, path)

	stdout, err := cmd.Output()
	exitErr := &exec.ExitError{}
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	return result{
		Stdout:   stdout,
		Stderr:   exitErr.Stderr,
		ExitCode: cmd.ProcessState.ExitCode(),
	}
}

func computeDiff(want, got []byte) string {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(got), string(want), false)
	color.NoColor = false
	return fmt.Sprint(color.RedString("got "), color.GreenString("want\n"), dmp.DiffPrettyText(diffs))
}

var printsRegexp = regexp.MustCompile(`// prints (.+)`)
var errorsRegexp = regexp.MustCompile(`// errors\n((?:// .+\n)*// .+)`)

func parseExpectedResult(t *testing.T, path string) result {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	r := result{
		Stdout: parseExpectedStdout(data),
		Stderr: parseExpectedStderr(t, path, data),
	}
	if len(r.Stderr) > 0 {
		r.ExitCode = 1
	}

	return r
}

func parseExpectedStdout(data []byte) []byte {
	var b bytes.Buffer
	for _, match := range printsRegexp.FindAllSubmatch(data, -1) {
		if !bytes.Equal(match[1], []byte("<empty>")) {
			b.Write(match[1])
		}
		b.WriteByte('\n')
	}
	return b.Bytes()
}

func parseExpectedStderr(t *testing.T, path string, data []byte) []byte {
	var b bytes.Buffer
	matches := errorsRegexp.FindAllSubmatch(data, -1)
	if len(matches) > 1 {
		t.Fatalf(`test files can only contain 1 "// errors" comment, %d found in %s`, len(matches), path)
	}
	for _, match := range matches {
		for _, line := range bytes.Split(match[1], []byte("\n")) {
			b.Write(bytes.TrimPrefix(line, []byte("// ")))
			b.WriteByte('\n')
		}
	}
	return b.Bytes()
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
	} else {
		t.Logf("stderr: <empty>")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	data = updateExpectedStdout(t, path, data, result.Stdout)
	data = updateExpectedStderr(t, path, data, result.Stderr)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func updateExpectedStdout(t *testing.T, path string, data []byte, stdout []byte) []byte {
	var lines [][]byte
	if len(stdout) > 0 {
		lines = bytes.Split(bytes.TrimSuffix(stdout, []byte("\n")), []byte("\n"))
	}
	locs := printsRegexp.FindAllSubmatchIndex(data, -1)
	if len(lines) != len(locs) {
		t.Fatalf(`%d "// prints" %s found in %s but %d %s printed to stdout, these should be equal`,
			len(locs), pluralise("comment", len(locs)), path, len(lines), pluralise("line", len(lines)))
	}
	if len(stdout) == 0 {
		return data
	}

	var b bytes.Buffer
	lastEnd := 0
	for i, loc := range locs {
		start, end := loc[2], loc[3]
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

func updateExpectedStderr(t *testing.T, path string, data []byte, stderr []byte) []byte {
	locs := errorsRegexp.FindAllSubmatchIndex(data, -1)

	switch len(locs) {
	case 0:
		if len(stderr) > 0 {
			t.Fatalf(`stderr not empty but no "// errors" comment found in %s`, path)
		}
		return data
	case 1:
		if len(stderr) == 0 {
			t.Fatalf(`"// errors" comment found in %s but nothing printed to stderr`, path)
		}
	default:
		t.Fatalf(`test files can only contain 1 "// errors" comment, %d found in %s`, len(locs), path)
	}

	start, end := locs[0][2], locs[0][3]
	var b bytes.Buffer
	b.Write(data[:start])

	stderrLines := bytes.SplitAfter(bytes.TrimSuffix(stderr, []byte("\n")), []byte("\n"))
	for _, line := range stderrLines {
		b.WriteString("// ")
		b.Write(line)
	}
	b.Write(data[end:])

	return b.Bytes()
}

func pluralise(s string, n int) string {
	if n == 1 {
		return s
	}
	return s + "s"
}
