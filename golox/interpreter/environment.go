package interpreter

import (
	"fmt"

	"github.com/marcuscaisey/lox/golox/token"
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
		panic(fmt.Sprintf("%s has already been declared", ident))
	}
	e.valuesByIdent[ident] = value
}

// Define defines an identifier in this environment and assigns it an initial value.
// If the identifier has already been defined in this environment, then a runtime error is raised.
// This method should be used for defining values which originated from an assignment in code. For example, a variable
// or function declaration. Otherwise, use Set.
func (e *environment) Define(tok token.Token, value loxObject) {
	if _, ok := e.valuesByIdent[tok.Literal]; ok {
		panic(newTokenRuntimeErrorf(tok, "%s has already been declared", tok.Literal))
	}
	e.valuesByIdent[tok.Literal] = value
}

// Assign assigns a value to a variable.
// If the variable has not been defined then a runtime error is raised.
func (e *environment) Assign(tok token.Token, value loxObject) {
	_, ok := e.valuesByIdent[tok.Literal]
	if !ok {
		panic(newTokenRuntimeErrorf(tok, "%s has not been declared", tok.Literal))
	}
	e.valuesByIdent[tok.Literal] = value
}

// AssignAt assigns a value to a variable in the environment distance levels up the parent chain.
func (e *environment) AssignAt(distance int, tok token.Token, value loxObject) {
	e.ancestor(distance).Assign(tok, value)
}

// Get returns the value of an identifier.
// If the identifier has not been defined in this environment or any of its parents, then a runtime error is raised.
func (e *environment) Get(tok token.Token) loxObject {
	value, ok := e.valuesByIdent[tok.Literal]
	if !ok {
		panic(newTokenRuntimeErrorf(tok, "%s has not been declared", tok.Literal))
	}
	if value == nil {
		panic(newTokenRuntimeErrorf(tok, "%s has not been defined", tok.Literal))
	}
	return value
}

// GetAt returns the value of an identifier in the environment distance levels up the parent chain.
func (e *environment) GetAt(distance int, tok token.Token) loxObject {
	return e.ancestor(distance).Get(tok)
}

func (e *environment) ancestor(n int) *environment {
	ancestor := e
	for range n {
		ancestor = ancestor.parent
		if ancestor == nil {
			panic(fmt.Sprintf("ancestor %d is out of range", n))
		}
	}
	return ancestor
}
