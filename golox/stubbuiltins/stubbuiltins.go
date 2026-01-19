// Package stubbuiltins provides the source code for stubs of Lox's built-ins and a function to parse them.
// Built-ins are not actually implemented in Lox, but these stubs allow tools to pretend that they are.
package stubbuiltins

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/parser"
)

//go:embed builtins.lox
var builtinsSrc []byte

//go:embed builtins_extra_features.lox
var builtinsExtraFeaturesSrc []byte

type config struct {
	extraFeatures bool
}

// Option can be passed to [MustParse] to configure its behaviour.
type Option func(*config)

// WithExtraFeatures enables extra features that https://github.com/marcuscaisey/lox implements but the base Lox
// language does not.
// Extra features are enabled by default.
func WithExtraFeatures(enabled bool) Option {
	return func(c *config) {
		c.extraFeatures = enabled
	}
}

// MustParse parses the stubs of Lox's built-ins and returns the declarations.
// filename is the name of the file that the declarations will be associated with.
func MustParse(filename string, opts ...Option) []ast.Decl {
	cfg := &config{extraFeatures: true}
	for _, opt := range opts {
		opt(cfg)
	}
	src := builtinsSrc
	if cfg.extraFeatures {
		src = builtinsExtraFeaturesSrc
	}
	program, err := parser.Parse(bytes.NewBuffer(src), filename, parser.WithComments(true))
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

// IsInternal reports whether a declaration is an internal stub declaration. These declarations should not be surfaced
// by tooling. They are marked with an "@internal" comment.
func IsInternal(decl ast.Decl) bool {
	documentedNode, ok := decl.(ast.Documented)
	return ok && documentedNode.Documentation() == "@internal"
}
