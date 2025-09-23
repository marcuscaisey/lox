// Entry point for jloxify.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"regexp"
)

var replacements = map[string]string{
	`(?m)^\[class ([A-Za-z_][A-Za-z0-9_]*)\]$`:                                              "Foo",
	`(?m)^\[([A-Za-z_][A-Za-z0-9_]*) object\]$`:                                             "$1 instance",
	`(?m)^\[(?:bound method [A-Za-z_][A-Za-z0-9_]*\.|function )([A-Za-z_][A-Za-z0-9_]*)\]$`: "<fn $1>",
	`(?m)^\[builtin function [A-Za-z_][A-Za-z0-9_]*\]$`:                                     "<native fn>",
}

type errorReplacement struct {
	Code     int
	Template string
}

var errorReplacements = map[string]errorReplacement{
	`^init\(\) cannot return a value$`:                         {65, "Error at 'return': Can't return a value from an initializer."},
	`^unterminated string literal$`:                            {65, "Error: Unterminated string."},
	`^expected expression$`:                                    {65, "Error at '$snippet': Expect expression."},
	`^cannot define more than 255 function parameters$`:        {65, "Error at '$snippet': Can't have more than 255 parameters."},
	`^'this' can only be used inside a method definition$`:     {65, "Error at 'this': Can't use 'this' outside of a class."},
	`^'return' can only be used inside a function definition$`: {65, "Error at 'return': Can't return from top-level code."},
	`^expected property name$`:                                 {65, "Error at '$snippet': Expect property name after '.'."},
	`^expected variable name$`:                                 {65, "Error at '$snippet': Expect variable name."},
	`^invalid assignment target$`:                              {65, "Error at '=': Invalid assignment target."},
	`^([A-Za-z_][A-Za-z0-9_]*) has already been declared$`:     {65, "Error at '$1': Already a variable with this name in this scope."},
	`^([A-Za-z_][A-Za-z0-9_]*) read in its own initialiser$`:   {65, "Error at '$1': Can't read local variable in its own initializer."},
	`^[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)?\(\) accepts (\d+) arguments? but (\d+) (?:was|were) given$`: {70, `Expected $1 arguments but got $2.`},
	`^'(?:<|<=|>|>=|-|/)' operator cannot be used with types '[A-Za-z_][A-Za-z0-9_]*' and '[A-Za-z_][A-Za-z0-9_]*'$`:  {70, "Operands must be numbers."},
	`^'-' operator cannot be used with type '[A-Za-z_][A-Za-z0-9_]*'$`:                                                {70, "Operand must be a number."},
	`^'\+' operator cannot be used with types '[A-Za-z_][A-Za-z0-9_]*' and '[A-Za-z_][A-Za-z0-9_]*'$`:                 {70, "Operands must be two numbers or two strings."},
	`^'[A-Za-z_][A-Za-z0-9_]*' object has no property ([A-Za-z_][A-Za-z0-9_]*)$`:                                      {70, "Undefined property '$1'."},
	`^([A-Za-z_][A-Za-z0-9_]*) has not been declared$`:                                                                {70, "Undefined variable '$1'."},
	`^'[A-Za-z_][A-Za-z0-9_]*' object is not callable$`:                                                               {70, "Can only call functions and classes."},
	`^property access is not valid for '[A-Za-z_][A-Za-z0-9_]*' object$`:                                              {70, "Only instances have properties."},
	`^property assignment is not valid for '[A-Za-z_][A-Za-z0-9_]*' object$`:                                          {70, "Only instances have fields."},
}

var (
	printHelp = flag.Bool("help", false, "Print this message")
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: jloxify <interpreter> <script>")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Options:")
	flag.PrintDefaults()
}

func exitWithUsageErr(msg string) {
	fmt.Fprintf(os.Stderr, "error: %s\n\n", msg)
	flag.Usage()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if *printHelp {
		flag.Usage()
		os.Exit(0)
	}

	switch len(flag.Args()) {
	case 0:
		exitWithUsageErr("interpreter and script arguments not provided")
	case 1:
		exitWithUsageErr("script argument not provided")
	}

	interpreter := flag.Arg(0)
	script := flag.Arg(1)
	if err := run(interpreter, script); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(interpreter string, script string) error {
	cmd := exec.Command(interpreter, script)
	stdout, err := cmd.Output()
	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		return err
	}

	newStdout := translateStdout(stdout)

	if err == nil {
		os.Stdout.Write(newStdout)
		return nil
	}

	code, newStderr, err := translateStderr(exitErr.Stderr)
	if err != nil {
		return err
	}
	os.Stdout.Write(newStdout)
	os.Stderr.WriteString(newStderr)
	os.Exit(code)

	return nil
}

func translateStdout(stdout []byte) []byte {
	for pattern, template := range replacements {
		re := regexp.MustCompile(pattern)
		stdout = re.ReplaceAll(stdout, []byte(template))
	}
	return stdout
}

func translateStderr(stderr []byte) (int, string, error) {
	errorRe := regexp.MustCompile(`(?m)^(\d+):\d+: error: (.+)\n(.+)\n(\s*~+)$`)
	match := errorRe.FindSubmatch(stderr)
	if len(match) == 0 {
		return 0, "", fmt.Errorf("translating stderr: error pattern %q not found:\n%s", errorRe, stderr)
	}

	msg := match[2]
	for msgPattern, replacement := range errorReplacements {
		msgRe := regexp.MustCompile(msgPattern)
		if !msgRe.Match(msg) {
			continue
		}

		highlightedLine := match[3]
		highlightLine := match[4]
		highlightStart := bytes.IndexRune(highlightLine, '~')
		snippet := highlightedLine[highlightStart:len(highlightLine)]
		template := bytes.ReplaceAll([]byte(replacement.Template), []byte("$snippet"), snippet)

		newMsg := msgRe.ReplaceAll(msg, template)

		line := match[1]
		var newStderr string
		switch replacement.Code {
		case 70:
			newStderr = fmt.Sprintf("%s\n[line %s]\n", newMsg, line)
		case 65:
			newStderr = fmt.Sprintf("[line %s] %s\n", line, newMsg)
		default:
			panic(fmt.Sprintf("unknown code: %d", replacement.Code))
		}
		return replacement.Code, newStderr, nil
	}

	return 0, "", fmt.Errorf("translating stderr: no replacement defined for error message %q", msg)
}
