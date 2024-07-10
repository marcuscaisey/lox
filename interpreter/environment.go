package interpreter

import (
	"fmt"

	"github.com/marcuscaisey/golox/token"
)

type environment struct {
	parent        *environment
	valuesByIdent map[string]loxObject
}

func newEnvironment() *environment {
	return &environment{
		valuesByIdent: make(map[string]loxObject),
	}
}

// Child creates a new child environment of this environment.
func (e *environment) Child() *environment {
	env := newEnvironment()
	env.parent = e
	return env
}

// Set assigns a value to an identifier in this environment.
// If the identifier has already been defined in this environment, then this method panics.
// This method should be used for defining values which did not originate from an assignment in code. For example,
// defining built-in functions or function arguments. Otherwise, use Define.
func (e *environment) Set(ident string, value loxObject) {
	if _, ok := e.valuesByIdent[ident]; ok {
		// It's a bug if we end up here
		panic(fmt.Sprintf("%s has already been defined", ident))
	}
	e.valuesByIdent[ident] = value
}

// Define defines an identifier in this environment and assigns it an initial value.
// If the identifier has already been defined in this environment, then a runtime error is raised.
// This method should be used for defining values which originated from an assignment in code. For example, a variable
// or function declaration. Otherwise, use Set.
func (e *environment) Define(tok token.Token, value loxObject) {
	if _, ok := e.valuesByIdent[tok.Literal]; ok {
		panic(newTokenRuntimeErrorf(tok, "%s has already been defined", tok.Literal))
	}
	e.valuesByIdent[tok.Literal] = value
}

// Assign assigns a value to a variable.
// If the variable has not been defined then a runtime error is raised.
func (e *environment) Assign(tok token.Token, value loxObject) {
	if e == nil {
		panic(newTokenRuntimeErrorf(tok, "%s has not been defined", tok.Literal))
	}

	if _, ok := e.valuesByIdent[tok.Literal]; ok {
		e.valuesByIdent[tok.Literal] = value
		return
	}

	e.parent.Assign(tok, value)
}

// Get returns the value of an identifier.
// If the identifier has not been defined in this environment or any of its parents, then a runtime error is raised.
func (e *environment) Get(tok token.Token) loxObject {
	if e == nil {
		panic(newTokenRuntimeErrorf(tok, "%s has not been defined", tok.Literal))
	}

	if value, ok := e.valuesByIdent[tok.Literal]; ok {
		if value == nil {
			panic(newTokenRuntimeErrorf(tok, "%s has not been initialised", tok.Literal))
		}
		return value
	}

	return e.parent.Get(tok)
}
