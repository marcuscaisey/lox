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
	"strings"
)

var replacements = map[string]string{
	`(?m)^\[class ([A-Za-z_][A-Za-z0-9_]*)\]$`:                                              "$1",
	`(?m)^\[([A-Za-z_][A-Za-z0-9_]*) object\]$`:                                             "$1 instance",
	`(?m)^\[(?:bound method [A-Za-z_][A-Za-z0-9_]*\.|function )([A-Za-z_][A-Za-z0-9_]*)\]$`: "<fn $1>",
	`(?m)^\[built-in function [A-Za-z_][A-Za-z0-9_]*\]$`:                                    "<native fn>",
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
	`^'super' can only be used inside a method definition$`:    {65, "Error at 'super': Can't use 'super' outside of a class."},
	`^'super' can only be used inside a subclass$`:             {65, "Error at 'super': Can't use 'super' in a class with no superclass."},
	`^'return' can only be used inside a function definition$`: {65, "Error at 'return': Can't return from top-level code."},
	`^expected property name$`:                                 {65, "Error at '$snippet': Expect property name after '.'."},
	`^expected variable name$`:                                 {65, "Error at '$snippet': Expect variable name."},
	`^invalid assignment target$`:                              {65, "Error at '=': Invalid assignment target."},
	`^'([A-Za-z_][A-Za-z0-9_]*)' has already been declared$`:   {65, "Error at '$1': Already a variable with this name in this scope."},
	`^'([A-Za-z_][A-Za-z0-9_]*)' read in its own initialiser$`: {65, "Error at '$1': Can't read local variable in its own initializer."},
	`^cannot pass more than 255 arguments to function$`:        {65, "Error at '$snippet': Can't have more than 255 arguments."},
	`^class cannot inherit from itself$`:                       {65, "Error at '$snippet': A class can't inherit from itself."},
	`^expected superclass name$`:                               {65, "Error at '$snippet': Expect superclass name."},
	`^[A-Za-z_][A-Za-z0-9_]*(?:\.[A-Za-z_][A-Za-z0-9_]*)?\(\) accepts (\d+) arguments? but (\d+) (?:was|were) given$`: {70, `Expected $1 arguments but got $2.`},
	`^'(?:<|<=|>|>=|-|/)' operator cannot be used with types '[A-Za-z_][A-Za-z0-9_]*' and '[A-Za-z_][A-Za-z0-9_]*'$`:  {70, "Operands must be numbers."},
	`^'-' operator cannot be used with type '[A-Za-z_][A-Za-z0-9_]*'$`:                                                {70, "Operand must be a number."},
	`^'\+' operator cannot be used with types '[A-Za-z_][A-Za-z0-9_]*' and '[A-Za-z_][A-Za-z0-9_]*'$`:                 {70, "Operands must be two numbers or two strings."},
	`^'[A-Za-z_][A-Za-z0-9_]*' object has no property '([A-Za-z_][A-Za-z0-9_]*)'$`:                                    {70, "Undefined property '$1'."},
	`^'([A-Za-z_][A-Za-z0-9_]*)' has not been declared$`:                                                              {70, "Undefined variable '$1'."},
	`^'[A-Za-z_][A-Za-z0-9_]*' value is not callable$`:                                                                {70, "Can only call functions and classes."},
	`^property access is not valid for '[A-Za-z_][A-Za-z0-9_]*' value$`:                                               {70, "Only instances have properties."},
	`^property assignment is not valid for '[A-Za-z_][A-Za-z0-9_]*' value$`:                                           {70, "Only instances have fields."},
	`^'[A-Za-z_][A-Za-z0-9_]*' class has no method '([A-Za-z_][A-Za-z0-9_]*)'$`:                                       {70, "Undefined property '$1'."},
	`^expected superclass to be a class, got '[a-z]+'$`:                                                               {70, "Superclass must be a class."},
}

func main() {
	os.Exit(cli())
}

type usageError string

func (e usageError) Error() string {
	return fmt.Sprintf("error: %s", string(e))
}

func cli() int {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: jloxify <interpreter> <script>")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
	printHelp := flag.Bool("help", false, "Print this message")

	flag.Parse()

	if *printHelp {
		flag.Usage()
		return 0
	}

	if err := jloxify(flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var usageErr usageError
		if errors.As(err, &usageErr) {
			fmt.Fprintln(os.Stderr)
			flag.Usage()
			return 2
		}
		return 1
	}

	return 0
}

func jloxify(args []string) error {
	switch len(flag.Args()) {
	case 0:
		return usageError("interpreter and script arguments not provided")
	case 1:
		return usageError("script argument not provided")
	}

	interpreter := args[0]
	script := args[1]
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
	matches := errorRe.FindAllSubmatch(stderr, -1)
	if len(matches) == 0 {
		return 0, "", fmt.Errorf("translating stderr: error pattern %q not found:\n%s", errorRe, stderr)
	}

	var code int
	var newStderrLines []string
	var msgs [][]byte
	for _, match := range matches {
		msg := match[2]
		msgs = append(msgs, msg)
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
			var newStderrLine string
			switch replacement.Code {
			case 70:
				newStderrLine = fmt.Sprintf("%s\n[line %s]\n", newMsg, line)
			case 65:
				newStderrLine = fmt.Sprintf("[line %s] %s\n", line, newMsg)
			default:
				panic(fmt.Sprintf("unknown code: %d", replacement.Code))
			}
			if code != 0 && replacement.Code != code {
				return 0, "", fmt.Errorf("translating stderr: conflicting codes for error messages %q", msgs)
			}
			code = replacement.Code
			newStderrLines = append(newStderrLines, newStderrLine)
		}
	}

	if len(newStderrLines) > 0 {
		return code, strings.Join(newStderrLines, ""), nil
	}

	return 0, "", fmt.Errorf("translating stderr: no replacements defined for error messages %q", msgs)
}
