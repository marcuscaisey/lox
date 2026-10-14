package loxtest

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var (
	printsRe    = regexp.MustCompile(`// prints: (.+)`)
	errorRe     = regexp.MustCompile(`// error: (.+)`)
	argumentsRe = regexp.MustCompile(`// arguments: (.+)`)
)

// InterpreterRunner is a [Runner] which tests a Lox interpreter using the comments described in
// test/README.md as expectations.
type InterpreterRunner struct {
	rootDir         string
	interpreterPath string
}

// NewInterpreterRunner creates a new [InterpreterRunner].
func NewInterpreterRunner(t *testing.T, interpreterPath string) *InterpreterRunner {
	rootDir := MustGoModuleRoot(t)
	return &InterpreterRunner{
		rootDir:         rootDir,
		interpreterPath: interpreterPath,
	}
}

// Test implements the [Runner] interface.
func (r *InterpreterRunner) Test(t *testing.T, path string) {
	fileContents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	want := r.parseExpectedResult(fileContents)
	scriptArgs := r.mustParseScriptArguments(t, path, fileContents)

	got := r.mustRunInterpreter(t, path, scriptArgs)

	if got.ExitCode != want.ExitCode {
		t.Fatalf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", got.ExitCode, want.ExitCode, got.Stdout, got.Stderr)
	}

	if diff := TextDiff(got.Stdout, want.Stdout); diff != "" {
		t.Errorf("incorrect output printed to stdout:\n%s\nstdout:\n%s", diff, got.Stdout)
	}

	if diff := LinesDiff(got.Errors, want.Errors); diff != "" {
		t.Errorf("incorrect errors printed to stderr:\n%s\nstderr:\n%s", diff, got.Stderr)
	}
}

type interpreterResult struct {
	Stdout   string
	Stderr   []byte
	Errors   []string
	ExitCode int
}

func (r *InterpreterRunner) mustRunInterpreter(t *testing.T, path string, scriptArgs []string) *interpreterResult {
	args := append([]string{path}, scriptArgs...)
	cmd := exec.Command(r.interpreterPath, args...)

	relInterpreterPath, err := filepath.Rel(r.rootDir, r.interpreterPath)
	if err != nil {
		t.Fatalf("making interpreter path relative: %s", err)
	}

	relPath, err := filepath.Rel(r.rootDir, path)
	if err != nil {
		t.Fatalf("making test file path relative: %s", err)
	}
	t.Logf("%s %s", relInterpreterPath, relPath)

	stdout, err := cmd.Output()

	exitErr := &exec.ExitError{}
	if err != nil && !errors.As(err, &exitErr) {
		t.Fatal(err)
	}
	var errors []string
	errorRe := regexp.MustCompile(`(?m)^\d+:(?:\d+:)? error: (.+)$`)
	for _, match := range errorRe.FindAllStringSubmatch(string(exitErr.Stderr), -1) {
		errors = append(errors, match[1])
	}

	return &interpreterResult{
		Stdout:   string(stdout),
		Stderr:   exitErr.Stderr,
		Errors:   errors,
		ExitCode: cmd.ProcessState.ExitCode(),
	}
}

func (r *InterpreterRunner) parseExpectedResult(fileContents []byte) *interpreterResult {
	stdoutLines := ParseComments(fileContents, printsRe)
	stdout := strings.Join(stdoutLines, "\n")
	if stdout != "" {
		stdout += "\n"
	}
	result := &interpreterResult{
		Stdout: stdout,
		Errors: ParseComments(fileContents, errorRe),
	}
	if len(result.Errors) > 0 {
		result.ExitCode = 1
	}

	return result
}

// Update implements the [Runner] interface.
func (r *InterpreterRunner) Update(t *testing.T, path string) {
	t.Logf("updating expected output for %s", path)

	fileContents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	scriptArgs := r.mustParseScriptArguments(t, path, fileContents)
	result := r.mustRunInterpreter(t, path, scriptArgs)

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
	contents = MustUpdateComments(t, path, contents, printsRe, stdoutLines)
	contents = MustUpdateComments(t, path, contents, errorRe, result.Errors)

	if err := os.WriteFile(path, contents, 0644); err != nil {
		t.Fatal(err)
	}
}

func (r *InterpreterRunner) mustParseScriptArguments(t *testing.T, path string, fileContents []byte) []string {
	argumentsLines := ParseComments(fileContents, argumentsRe)
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
