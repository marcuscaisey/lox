// Entry point for typegen.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"go/format"
	"os"
	"slices"
	"strings"

	"github.com/marcuscaisey/lox/loxls/lsp/protocol/typegen/generate"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol/typegen/metamodel"
)

const methodCommentDirective = "//typegen:method"

func main() {
	os.Exit(cli())
}

type usageError string

func (e usageError) Error() string {
	return fmt.Sprintf("error: %s", string(e))
}

func cli() int {
	flag.Usage = func() {
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

Usage: typegen [options] [<method>...]

Options:
`), methodCommentDirective)
		flag.PrintDefaults()
	}
	lspVersion := flag.String("lsp-version", "3.17", "LSP version")
	pkg := flag.String("package", "protocol", "Package the file will belong to")
	output := flag.String("output", "protocol.go", "Output file")

	flag.Parse()

	if err := typeGen(flag.Args(), *lspVersion, *pkg, *output); err != nil {
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

func typeGen(args []string, lspVersion string, pkg string, output string) error {
	methodComments, err := parseMethodComments()
	if err != nil {
		return err
	}
	if len(args) > 0 && len(methodComments) > 0 {
		return usageError(fmt.Sprintf("cannot specify methods as arguments and via %s comments", methodCommentDirective))
	}
	methods := append(slices.Clone(args), methodComments...)

	if len(methods) == 0 {
		flag.Usage()
		return nil
	}

	metaModel, err := metamodel.Load(lspVersion)
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

	src := generate.Source(types, metaModel, pkg)

	formattedSrc, err := format.Source([]byte(src))
	if err != nil {
		return fmt.Errorf("formatting generated file: %s\ncontents: %s", err, src)
	}

	return os.WriteFile(output, formattedSrc, 0644)
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
