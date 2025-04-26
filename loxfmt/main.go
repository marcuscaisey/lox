// Entry point for the loxfmt Lox formatter.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/format"
	"github.com/marcuscaisey/lox/golox/parser"
)

var (
	write    = flag.Bool("w", false, "Write result to (source) file instead of stdout")
	printAST = flag.Bool("p", false, "Print the AST only")
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: loxfmt [flags] [path]\n")
	fmt.Fprintf(flag.CommandLine.Output(), "\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
	flag.PrintDefaults()
}

func exitWithUsageErr(msg string) {
	fmt.Fprintf(flag.CommandLine.Output(), "error: %s\n\n", msg)
	flag.Usage()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()

	if len(flag.Args()) > 1 {
		exitWithUsageErr("at most one path can be provided")
	}

	path := flag.Arg(0)

	if path == "" && *write {
		exitWithUsageErr("error: cannot use -w with standard input")
	}

	if err := run(flag.Arg(0)); err != nil {
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

	program, err := parser.Parse(reader, path, parser.WithComments(true))
	if *printAST {
		ast.Print(program)
		return err
	}
	if err != nil {
		return err
	}

	formatted := format.Node(program)
	if *write {
		if err := os.WriteFile(path, []byte(formatted), 0644); err != nil {
			return fmt.Errorf("failed to write formatted source to file: %w", err)
		}
	} else {
		fmt.Print(formatted)
	}

	return nil
}
