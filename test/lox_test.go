package test

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"github.com/fatih/color"
	"github.com/google/go-cmp/cmp"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

var update = flag.Bool("update", false, "updates the expected output of each test")

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
	for _, part := range strings.Split(s, "_") {
		r, size := utf8.DecodeRuneInString(part)
		b.WriteRune(unicode.ToUpper(r))
		b.WriteString(part[size:])
	}
	return b.String()
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

func computeTextDiff(want, got string) string {
	color.NoColor = false
	edits := myers.ComputeEdits(span.URIFromPath("want"), want, got)
	diff := fmt.Sprint(gotextdiff.ToUnified("want", "got", want, edits))
	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "-") {
			lines[i] = color.GreenString(line)
		} else if strings.HasPrefix(line, "+") {
			lines[i] = color.RedString(line)
		}
	}
	return strings.Join(lines, "\n")
}
