// Package analyse implements static analysis of Lox programs.
package analyse

import (
	"slices"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerr"
)

// Option can be passed to [Program] and [ResolveIdents] to configure analysis behaviour.
type Option func(*config)

type config struct {
	replMode bool
}

// WithREPLMode configures identifiers to be resolved in REPL mode.
// In REPL mode, the following identifier checks are disabled:
//   - declared and never used
//   - declared more than once in the same scope
//   - used before they are declared
func WithREPLMode(enabled bool) Option {
	return func(i *config) {
		i.replMode = enabled
	}
}

// Program performs static analysis of a program and reports any errors detected.
// The analyses performed are described in the doc comments for [ResolveIdents] and [CheckSemantics].
func Program(program *ast.Program, builtins []ast.Decl, opts ...Option) loxerr.Errors {
	_, resolveErrs := ResolveIdents(program, builtins, opts...)
	semanticErrs := CheckSemantics(program)
	return slices.Concat(resolveErrs, semanticErrs)
}
