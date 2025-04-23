// Package stubbuiltins provides the source code for stubs of Lox's built-ins and a function to parse them.
// Built-ins are not actually implemented in Lox, but these stubs allow tools to pretend that they are.
package stubbuiltins

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/marcuscaisey/lox/lox/ast"
	"github.com/marcuscaisey/lox/lox/parser"
)

// Source is the source code for stubs of Lox's built-ins.
//
//go:embed builtins.lox
var Source []byte

// MustParse parses the stubs of Lox's built-ins and returns the declarations.
// filename is the name of the file that the declarations will be associated with.
func MustParse(filename string) []ast.Decl {
	program, err := parser.Parse(bytes.NewBuffer(Source), filename, parser.WithComments(true))
	if err != nil {
		panic(fmt.Sprintf("parsing built-in stubs: %s", err))
	}

	var decls []ast.Decl
	for _, stmt := range program.Stmts {
		if decl, ok := stmt.(ast.Decl); ok {
			decls = append(decls, decl)
		}
	}

	return decls
}
