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
	"github.com/marcuscaisey/lox/golox/loxerr"
	"github.com/marcuscaisey/lox/golox/parser"
	"github.com/marcuscaisey/lox/golox/stubbuiltins"
)

var (
	printHelp = flag.Bool("help", false, "Print this message")
)

func usage() {
	fmt.Fprintln(os.Stderr, "Usage: loxlint [flags] [path]")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "If no path is provided, the file is read from stdin.")
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

	if len(flag.Args()) > 1 {
		exitWithUsageErr("at most one path can be provided")
	}

	path := flag.Arg(0)
	if err := run(path); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(path string) error {
	var reader io.Reader = os.Stdin
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	program, err := parser.Parse(reader, path)
	if err != nil {
		return err
	}

	builtins := stubbuiltins.MustParse("builtins.lox")
	err = analyse.Program(program, builtins)
	var loxErrs loxerr.Errors
	errors.As(err, &loxErrs)
	if err := loxErrs.NonFatal().Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return nil
}
