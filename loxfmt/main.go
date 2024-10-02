// Entry point for the loxfmt Lox formatter.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/parser"
)

var (
	write    = flag.Bool("w", false, "Write result to (source) file instead of stdout")
	printAST = flag.Bool("p", false, "Print the AST only")
)

// nolint:revive
func Usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: loxfmt [flags] [path]\n")
	fmt.Fprintf(flag.CommandLine.Output(), "\n")
	fmt.Fprintf(flag.CommandLine.Output(), "Options:\n")
	flag.PrintDefaults()
}

func main() {
	log.SetFlags(0)

	flag.Usage = Usage
	flag.Parse()

	if len(flag.Args()) != 1 {
		flag.Usage()
		os.Exit(2)
	}

	if err := run(flag.Arg(0)); err != nil {
		log.Fatal(err)
	}
}

func run(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	reader := newNamedReader(bytes.NewReader(data), path)

	program, err := parser.Parse(reader, parser.WithComments())
	if *printAST {
		ast.Print(program)
		return err
	}
	if err != nil {
		return err
	}

	formatted := format(program)
	if *write {
		if err := os.WriteFile(path, []byte(formatted), 0644); err != nil {
			return fmt.Errorf("failed to write formatted source to file: %w", err)
		}
	} else {
		fmt.Print(formatted)
	}

	return nil
}

type namedReader struct {
	io.Reader
	name string
}

func newNamedReader(r io.Reader, name string) io.Reader {
	return namedReader{Reader: r, name: name}
}

func (n namedReader) Name() string {
	return n.name
}
