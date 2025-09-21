package test

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

func init() {
	ansi.Enabled = true
}

var (
	pwd    = flag.String("pwd", "", "directory that the test was invoked from")
	update = flag.Bool("update", false, "updates the expected output of each test")

	syntaxErrorComment = "// syntaxerror"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if *pwd == "" {
		fmt.Fprintln(os.Stderr, "-pwd flag must be provided")
		os.Exit(2)
	}
	os.Exit(m.Run())
}

type testRunner interface {
	Test(t *testing.T, path string)
	Update(t *testing.T, path string)
}

func runTests(t *testing.T, runner testRunner, path string) {
	matches, err := filepath.Glob(filepath.Join(path, "*"))
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range matches {
		testName := snakeToPascalCase(filepath.Base(path))
		if filepath.Ext(path) == ".lox" {
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
				runTests(t, runner, path)
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

func computeDiff(want, got any) string {
	diff := cmp.Diff(want, got, cmp.Transformer("BytesToString", func(b []byte) string {
		return string(b)
	}))
	return ansi.Sprintf("${GREEN}want -\n${RED}got +${DEFAULT}\n%s", colouriseDiff(diff))
}

func computeTextDiff(want, got string) string {
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
