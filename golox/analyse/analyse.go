// Package analyse implements static analysis of Lox programs.
package analyse

import (
	"errors"
	"iter"
	"slices"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerr"
)

// Option can be passed to [Program], [ResolveIdents], and [CheckSemantics] to configure analysis behaviour.
type Option func(*config)

type config struct {
	fatalOnly     bool
	extraFeatures bool
}

func newConfig(opts []Option) *config {
	cfg := &config{extraFeatures: true}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}

// WithFatalOnly configures only fatal errors to be reported.
func WithFatalOnly(enabled bool) Option {
	return func(cfg *config) {
		cfg.fatalOnly = enabled
	}
}

// WithExtraFeatures enables extra features that https://github.com/marcuscaisey/lox implements but the base Lox
// language does not.
// Extra features are enabled by default.
func WithExtraFeatures(enabled bool) Option {
	return func(c *config) {
		c.extraFeatures = enabled
	}
}

// Program performs static analysis of a program and reports any errors detected.
// builtins is a list of built-in declarations which are available in the global scope.
// The analyses performed are described in the doc comments for [ResolveIdents] and [CheckSemantics].
// If there is an error, it will be of type [loxerr.Errors].
func Program(program *ast.Program, builtins []ast.Decl, opts ...Option) error {
	_, resolveErr := ResolveIdents(program, builtins, opts...)
	semanticsErr := CheckSemantics(program, opts...)
	var resolveLoxErrs, semanticsLoxErrs loxerr.Errors
	errors.As(resolveErr, &resolveLoxErrs)
	errors.As(semanticsErr, &semanticsLoxErrs)
	loxErrs := slices.Concat(resolveLoxErrs, semanticsLoxErrs)
	return loxErrs.Err()
}

// InheritanceChain returns an iterator over the chain of classes used to look up possibly inherited properties.
// Iteration starts from the given class declaration, then successive iterations traverse its superclasses.
// identBindings is used to superclass identifiers to their declarations. This will typically be the result of
// [ResolveIdents].
func InheritanceChain(decl *ast.ClassDecl, identBindings map[*ast.Ident][]ast.Binding) iter.Seq[*ast.ClassDecl] {
	return func(yield func(*ast.ClassDecl) bool) {
		curClassDecl := decl
		for {
			if !yield(curClassDecl) {
				return
			}
			superclassBindings, ok := identBindings[curClassDecl.Superclass]
			if !ok {
				break
			}
			superclassDecl, ok := superclassBindings[0].(*ast.ClassDecl)
			if !ok {
				break
			}
			// An invalid class declaration might specify itself as its own superclass.
			if superclassDecl == curClassDecl {
				break
			}
			curClassDecl = superclassDecl
		}
	}
}
