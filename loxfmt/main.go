// Entry point for the loxfmt Lox formatter.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/parser"
	"github.com/marcuscaisey/lox/loxfmt/format"
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
		fmt.Fprintln(os.Stderr, "Usage: loxfmt [flags] [path]")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "If no path is provided, the file is read from stdin.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
	write := flag.Bool("write", false, "Write result to (source) file instead of stdout")
	printAST := flag.Bool("ast", false, "Print the AST")
	printHelp := flag.Bool("help", false, "Print this message")

	flag.Parse()

	if *printHelp {
		flag.Usage()
		return 0
	}

	if err := loxfmt(flag.Args(), *write, *printAST); err != nil {
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

func loxfmt(args []string, write bool, printAST bool) error {
	if len(args) > 1 {
		return usageError("at most one path can be provided")
	}
	if len(args) == 0 && write {
		return usageError("cannot use -write with standard input")
	}

	reader := io.Reader(os.Stdin)
	filename := "stdin"
	if len(args) > 0 {
		path := args[0]
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	program, err := parser.Parse(reader, filename, parser.WithComments(true))
	if printAST {
		ast.Print(program)
		return err
	}
	if err != nil {
		return err
	}

	formatted := format.Node(program)
	if write {
		if err := os.WriteFile(filename, []byte(formatted), 0644); err != nil {
			return fmt.Errorf("failed to write formatted source to file: %w", err)
		}
	} else {
		fmt.Print(formatted)
	}

	return nil
}
