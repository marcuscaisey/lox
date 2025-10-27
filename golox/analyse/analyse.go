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
	fatalOnly bool
}

// WithFatalOnly configures only fatal errors to be reported.
func WithFatalOnly(enabled bool) Option {
	return func(cfg *config) {
		cfg.fatalOnly = enabled
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
