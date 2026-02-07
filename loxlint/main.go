// Entry point for the loxlint Lox linter.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/marcuscaisey/lox/golox/analyse"
	"github.com/marcuscaisey/lox/golox/builtins"
	"github.com/marcuscaisey/lox/golox/parser"
)

func main() {
	os.Exit(cli())
}

type usageError string

func (e usageError) Error() string {
	return fmt.Sprintf("error: %s", string(e))
}

func cli() int {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: loxlint [options] [<path>]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "If no path is provided, the file is read from stdin.")
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

	if err := loxlint(flag.Args()); err != nil {
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

func loxlint(args []string) error {
	if len(args) > 1 {
		return usageError("at most one path can be provided")
	}

	filename := "<stdin>"
	reader := io.Reader(os.Stdin)
	if len(args) > 0 {
		filename := args[0]
		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	program, err := parser.Parse(reader, filename)
	if err != nil {
		return err
	}

	builtins := builtins.MustParseStubs("builtins.lox")
	return analyse.Program(program, builtins)
}
