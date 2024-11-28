// Entry point for typegen.
package main

import (
	"flag"
	"fmt"
	"go/format"
	"os"
	"strings"

	"github.com/marcuscaisey/lox/loxls/lsp/protocol/typegen/generate"
	"github.com/marcuscaisey/lox/loxls/lsp/protocol/typegen/metamodel"
)

var (
	lspVersion = flag.String("lsp-version", "3.17", "LSP version")
	pkg        = flag.String("package", "protocol", "Package the file will belong to")
	output     = flag.String("output", "protocol.go", "Output file")
)

func usage() {
	fmt.Fprintln(os.Stderr, strings.TrimSpace(`
typegen generates a Go file containing the types required to implement handlers
for the given LSP methods.

Usage: typegen [options] method ...

Options:
`))
	flag.PrintDefaults()
}

func main() {
	flag.Usage = usage
	flag.Parse()

	methods := flag.Args()

	if len(methods) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	if err := typeGen(methods, *lspVersion, *pkg, *output); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(2)
	}
}

func typeGen(methods []string, lspVersion string, pkg string, ouptut string) error {
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

	return os.WriteFile(ouptut, formattedSrc, 0644)
}
