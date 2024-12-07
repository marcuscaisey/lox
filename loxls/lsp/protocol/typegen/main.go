// Entry point for typegen.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/format"
	"os"
	"slices"
	"strings"

	"github.com/marcuscaisey/lox/loxls/lsp/protocol/typegen/generate"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol/typegen/metamodel"
)

var (
	lspVersion = flag.String("lsp-version", "3.17", "LSP version")
	pkg        = flag.String("package", "protocol", "Package the file will belong to")
	output     = flag.String("output", "protocol.go", "Output file")
)

const methodCommentDirective = "//typegen:method"

func usage() {
	fmt.Fprintf(os.Stderr, strings.TrimSpace(`
typegen generates a Go file containing the types required to implement handlers
for the given LSP methods.

Methods can either be specified as arguments or if invoked via go generate then
via "%[1]s" comments in the file containing the "//go:generate"
comment.

	package protocol
	//go:generate typegen
	%[1]s initialize
	%[1]s initialized
	%[1]s shutdown
	%[1]s exit

Usage: typegen [options] [method ...]

Options:
`), methodCommentDirective)
	flag.PrintDefaults()
}

func main() {
	if err := typeGen(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(2)
	}
}

func typeGen() error {
	flag.Usage = usage
	flag.Parse()

	methodArgs := flag.Args()
	methodComments, err := parseMethodComments()
	if err != nil {
		return err
	}
	if len(methodArgs) > 0 && len(methodComments) > 0 {
		return fmt.Errorf("cannot specify methods as arguments and via %s comments", methodCommentDirective)
	}
	methods := append(slices.Clone(methodArgs), methodComments...)

	if len(methods) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	metaModel, err := metamodel.Load(*lspVersion)
	if err != nil {
		return err
	}

	types, err := metaModel.MethodTypes(methods)
	if err != nil {
		return err
	}

	types = append(types, &metamodel.Type{
		Value: metamodel.ReferenceType{
			Kind: "",
			Name: "ErrorCodes",
		},
	})

	src := generate.Source(types, metaModel, *pkg)

	formattedSrc, err := format.Source([]byte(src))
	if err != nil {
		return fmt.Errorf("formatting generated file: %s\ncontents: %s", err, src)
	}

	return os.WriteFile(*output, formattedSrc, 0644)
}

func parseMethodComments() ([]string, error) {
	filename := os.Getenv("GOFILE")
	if filename == "" {
		return nil, nil
	}

	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("parsing %s comments: %s", methodCommentDirective, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var methods []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, methodCommentDirective+" ") {
			methods = append(methods, strings.TrimSpace(strings.TrimPrefix(line, methodCommentDirective)))
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parsing %s comments from %s: %s", methodCommentDirective, filename, err)
	}

	return methods, nil
}
