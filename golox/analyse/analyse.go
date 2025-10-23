// Package analyse implements static analysis of Lox programs.
package analyse

import (
	"errors"
	"slices"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerr"
)

// Option can be passed to [Program] and [ResolveIdents] to configure analysis behaviour.
type Option func(*config)

type config struct {
	replMode  bool
	fatalOnly bool
}

// WithREPLMode configures identifiers to be resolved in REPL mode.
// In REPL mode, the following identifier checks are disabled:
//   - declared and never used
//   - declared more than once in the same scope
//   - used before they are declared
func WithREPLMode(enabled bool) Option {
	// TODO: maybe we can get rid of this after removing most of the fatal errors from ResolveIdents?
	return func(i *config) {
		i.replMode = enabled
	}
}

// WithFatalOnly configures only fatal errors to be reported.
func WithFatalOnly(enabled bool) Option {
	return func(i *config) {
		i.fatalOnly = enabled
	}
}

// Program performs static analysis of a program and reports any errors detected.
// The analyses performed are described in the doc comments for [ResolveIdents] and [CheckSemantics].
// If there is an error, it will be of type [loxerr.Errors].
func Program(program *ast.Program, builtins []ast.Decl, opts ...Option) error {
	_, resolveErr := ResolveIdents(program, builtins, opts...)
	semanticsErr := CheckSemantics(program)
	var resolveLoxErrs, semanticsLoxErrs loxerr.Errors
	errors.As(resolveErr, &resolveLoxErrs)
	errors.As(semanticsErr, &semanticsLoxErrs)
	loxErrs := slices.Concat(resolveLoxErrs, semanticsLoxErrs)
	return loxErrs.Err()
}
