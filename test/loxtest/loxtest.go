// Package loxtest implements utilities for testing lox tools on the corpus of lox files defined under test/testdata.
package loxtest

import (
	"bytes"
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

	"github.com/google/go-cmp/cmp"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"

	"github.com/marcuscaisey/lox/golox/ansi"
)

var (
	update = flag.Bool("update", false, "updates the expected output of each test")

	syntaxErrorComment = "// syntaxerror"
)

func init() {
	ansi.Enabled = true
}

// Option can be passed to [Run] to configure its behaviour.
type Option func(*config)

// WithSkipSyntaxErrors configures whether files beginning with a `// syntaxerror` comment will be skipped. These
// comments mark that a file has syntax errors which cause it to be unparsable.
func WithSkipSyntaxErrors(enabled bool) Option {
	return func(c *config) {
		c.SkipSyntaxErrors = enabled
	}
}

// Runner defines how a test will be run or updated.
type Runner interface {
	// Test runs the test. It's passed the .lox file being tested and is responsible for failing the passed in
	// [*testing.T] if there are any errors.
	Test(t *testing.T, path string)
	// Test updates the expected output of the test. It's passed the .lox file being updated and is responsible for
	// failing the passed in [*testing.T] if there are any errors.
	Update(t *testing.T, path string)
}

// Run runs or updates a test for each .lox file under test/testdata. The provided runner defines how each test is run
// or updated.
// By default, [Runner.Test] is called in a subtest for each file. If the -update flag is passed to the test binary,
// then [Runner.Update] is called instead.
// All subtests are run in parallel.
func Run(t *testing.T, runner Runner, opts ...Option) {
	cfg := &config{
		SkipSyntaxErrors: true,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	rootDir := mustGoModuleRoot(t)
	testdataDir := filepath.Join(rootDir, "test", "testdata")
	run(t, runner, testdataDir, cfg)
}

type config struct {
	SkipSyntaxErrors bool
}

func run(t *testing.T, runner Runner, path string, cfg *config) {
	matches, err := filepath.Glob(filepath.Join(path, "*"))
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range matches {
		testName := snakeToPascalCase(filepath.Base(path))
		if filepath.Ext(path) == ".lox" {
			if cfg.SkipSyntaxErrors {
				contents, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if bytes.HasPrefix(contents, []byte(syntaxErrorComment)) {
					continue
				}
			}

			testName = strings.TrimSuffix(testName, ".lox")
			t.Run(testName, func(t *testing.T) {
				t.Parallel()
				if *update {
					runner.Update(t, path)
				} else {
					runner.Test(t, path)
				}
			})

		} else {
			t.Run(testName, func(t *testing.T) {
				t.Parallel()
				run(t, runner, path, cfg)
			})
		}

	}
}

func snakeToPascalCase(s string) string {
	var b strings.Builder
	for part := range strings.SplitSeq(s, "_") {
		r, size := utf8.DecodeRuneInString(part)
		b.WriteRune(unicode.ToUpper(r))
		b.WriteString(part[size:])
	}
	return b.String()
}

// ComputeDiff returns a human-readable report of the differences between a wanted and got value.
func ComputeDiff(want, got any) string {
	diff := cmp.Diff(want, got, cmp.Transformer("BytesToString", func(b []byte) string {
		return string(b)
	}))
	return ansi.Sprintf("${GREEN}want -\n${RED}got +${DEFAULT}\n%s", colouriseDiff(diff))
}

// ComputeTextDiff returns a human-readable report of the differences between a wanted and got string.
// If there are no differences, an empty string is returned.
// The output of this function is more readable than [ComputeDiff] for string inputs.
func ComputeTextDiff(want, got string) string {
	edits := myers.ComputeEdits(span.URIFromPath("want"), want, got)
	diff := fmt.Sprint(gotextdiff.ToUnified("want", "got", want, edits))
	return colouriseDiff(diff)
}

func colouriseDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "-") {
			lines[i] = ansi.Sprint("${GREEN}", line, "${DEFAULT}")
		} else if strings.HasPrefix(line, "+") {
			lines[i] = ansi.Sprint("${RED}", line, "${DEFAULT}")
		}
	}
	return strings.Join(lines, "\n")
}

// MustBuildBinary builds a Go binary defined in the github.com/marcuscaisey/lox Go module and returns the path to it.
// name should be a directory in the root of the module. A binary of the same name is output to the build directory.
func MustBuildBinary(t *testing.T, name string) string {
	t.Helper()

	rootDir := mustGoModuleRoot(t)
	buildDir := filepath.Join(rootDir, "build")
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		t.Fatalf("building %s: %s", name, err)
	}

	loxfmtPath := filepath.Join(buildDir, name)
	cmd := exec.Command("go", "build", "-o", loxfmtPath, "github.com/marcuscaisey/lox/"+name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("building %s: %s: %v\nOutput:\n%s\n", name, cmd.String(), err, string(output))
	}

	return loxfmtPath
}

func mustGoModuleRoot(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("determining go module root: %s", err)
	}

	for d := wd; d != "/"; d = filepath.Dir(d) {
		gomodPath := filepath.Join(d, "go.mod")
		if info, err := os.Stat(gomodPath); err == nil && !info.IsDir() {
			return d
		}
	}

	t.Fatal("determining go module root: no parent directory containing go.mod found")
	return ""
}

// ParseComments parses the comments of a file matching the given pattern.
func ParseComments(fileContents []byte, commentPattern *regexp.Regexp) [][]byte {
	var lines [][]byte
	for _, match := range commentPattern.FindAllSubmatch(fileContents, -1) {
		line := match[1]
		if bytes.Equal(match[1], []byte("<empty>")) {
			line = []byte{}
		}
		lines = append(lines, line)
	}
	return lines
}

// MustUpdateComments updates the comments of a file matching the given pattern with the contents of the given lines.
func MustUpdateComments(t *testing.T, filePath string, fileContents []byte, commentPattern *regexp.Regexp, lines [][]byte) []byte {
	matches := commentPattern.FindAllSubmatchIndex(fileContents, -1)
	if len(lines) != len(matches) {
		t.Fatalf(`%d "%s" %s found in %s but %d %s output, these should be equal`,
			len(matches), commentPattern, pluralise("comment", len(matches)), filePath, len(lines), pluralise("line", len(lines)))
	}
	if len(lines) == 0 {
		return fileContents
	}

	var b bytes.Buffer
	lastEnd := 0
	for i, match := range matches {
		start, end := match[2], match[3]
		b.Write(fileContents[lastEnd:start])
		if bytes.Equal(lines[i], []byte("")) {
			b.WriteString("<empty>")
		} else {
			b.Write(lines[i])
		}
		lastEnd = end
	}
	b.Write(fileContents[lastEnd:])

	return b.Bytes()
}

func pluralise(s string, n int) string {
	if n == 1 {
		return s
	}
	return s + "s"
}
