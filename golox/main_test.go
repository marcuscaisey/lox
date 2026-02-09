package main_test

import (
	"errors"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/marcuscaisey/lox/test/loxtest"
)

var (
	interpreter = flag.String("interpreter", "", "Interpreter to run tests against instead of golox")

	printsRe      = regexp.MustCompile(`// prints: (.+)`)
	errorRe       = regexp.MustCompile(`// error: (.+)`)
	argumentsRe   = regexp.MustCompile(`// arguments: (.+)`)
	environmentRe = regexp.MustCompile(`// environment: ((?:[^\s=]+=[^\s=]*\s*)+)`)
)

func TestGolox(t *testing.T) {
	rootDir := loxtest.MustGoModuleRoot(t)
	goloxPath := *interpreter
	if goloxPath == "" {
		goloxPath = loxtest.MustBuildBinary(t, "golox")
	}
	runner := newRunner(rootDir, goloxPath)
	loxtest.Run(t, runner, loxtest.WithSkipSyntaxErrors(false))
}

func newRunner(rootDir string, goloxPath string) *runner {
	return &runner{
		rootDir:   rootDir,
		goloxPath: goloxPath,
	}
}

type runner struct {
	rootDir   string
	goloxPath string
}

func (r *runner) Test(t *testing.T, path string) {
	fileContents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := r.parseExpectedResult(fileContents)
	scriptArgs := r.mustParseScriptArguments(t, path, fileContents)
	env := r.mustParseEnv(t, path, fileContents)

	got := r.mustRunGolox(t, path, scriptArgs, env)

	if got.ExitCode != want.ExitCode {
		t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", got.ExitCode, want.ExitCode, got.Stdout, got.Stderr)
	}

	if diff := loxtest.TextDiff(got.Stdout, want.Stdout); diff != "" {
		t.Errorf("incorrect output printed to stdout:\n%s\nstdout:\n%s", diff, got.Stdout)
	}

	if diff := loxtest.LinesDiff(got.Errors, want.Errors); diff != "" {
		t.Errorf("incorrect errors printed to stderr:\n%s\nstderr:\n%s", diff, got.Stderr)
	}
}

type goloxResult struct {
	Stdout   string
	Stderr   []byte
	Errors   []string
	ExitCode int
}

func (r *runner) mustRunGolox(t *testing.T, path string, scriptArgs []string, env []string) *goloxResult {
	args := append([]string{path}, scriptArgs...)
	cmd := exec.Command(r.goloxPath, args...)
	cmd.Env = env

	relPath, err := filepath.Rel(r.rootDir, path)
	if err != nil {
		t.Fatalf("making test file path relative: %s", err)
	}
	t.Logf("go run ./golox %s", relPath)

	stdout, err := cmd.Output()

	exitErr := &exec.ExitError{}
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	var errors []string
	errorRe := regexp.MustCompile(`(?m)^\d+:\d+: error: (.+)$`)
	for _, match := range errorRe.FindAllStringSubmatch(string(exitErr.Stderr), -1) {
		errors = append(errors, match[1])
	}

	return &goloxResult{
		Stdout:   string(stdout),
		Stderr:   exitErr.Stderr,
		Errors:   errors,
		ExitCode: cmd.ProcessState.ExitCode(),
	}
}

func (r *runner) parseExpectedResult(fileContents []byte) *goloxResult {
	stdoutLines := loxtest.ParseComments(fileContents, printsRe)
	stdout := strings.Join(stdoutLines, "\n")
	if stdout != "" {
		stdout += "\n"
	}
	result := &goloxResult{
		Stdout: stdout,
		Errors: loxtest.ParseComments(fileContents, errorRe),
	}
	if len(result.Errors) > 0 {
		result.ExitCode = 1
	}

	return result
}

func (r *runner) Update(t *testing.T, path string) {
	t.Logf("updating expected output for %s", path)

	fileContents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	scriptArgs := r.mustParseScriptArguments(t, path, fileContents)
	env := r.mustParseEnv(t, path, fileContents)
	result := r.mustRunGolox(t, path, scriptArgs, env)

	t.Logf("exit code: %d", result.ExitCode)
	if len(result.Stdout) > 0 {
		t.Logf("stdout:\n%s", result.Stdout)
	} else {
		t.Logf("stdout: <empty>")
	}
	if len(result.Stderr) > 0 {
		t.Logf("stderr:\n%s", result.Stderr)
		if len(result.Errors) > 0 {
			t.Logf("errors:\n%s", strings.Join(result.Errors, "\n"))
		} else {
			t.Logf("errors: <empty>")
		}
	} else {
		t.Logf("stderr: <empty>")
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	stdout := strings.TrimSuffix(result.Stdout, "\n")
	var stdoutLines []string
	if stdout != "" {
		stdoutLines = strings.Split(stdout, "\n")
	}
	contents = loxtest.MustUpdateComments(t, path, contents, printsRe, stdoutLines)
	contents = loxtest.MustUpdateComments(t, path, contents, errorRe, result.Errors)

	if err := os.WriteFile(path, contents, 0644); err != nil {
		t.Fatal(err)
	}
}

func (r *runner) mustParseScriptArguments(t *testing.T, path string, fileContents []byte) []string {
	argumentsLines := loxtest.ParseComments(fileContents, argumentsRe)
	var scriptArgs []string
	switch len(argumentsLines) {
	case 0:
		break
	case 1:
		scriptArgs = strings.Fields(argumentsLines[0])
	default:
		t.Fatalf("%d %q lines found in %s. There should be at most one.", len(argumentsLines), argumentsRe, path)
	}
	return scriptArgs
}

func (r *runner) mustParseEnv(t *testing.T, path string, fileContents []byte) []string {
	environmentLines := loxtest.ParseComments(fileContents, environmentRe)
	var env []string
	switch len(environmentLines) {
	case 0:
		break
	case 1:
		env = strings.Fields(environmentLines[0])
	default:
		t.Fatalf("%d %q lines found in %s. There should be at most one.", len(environmentLines), environmentRe, path)
	}
	return env
}
