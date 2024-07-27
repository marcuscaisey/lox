package interpreter

import (
	"fmt"

	"github.com/marcuscaisey/lox/golox/lox"
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

// Declare declares an identifier in this environment.
// If the identifier has already been declared in this environment, then a runtime error is raised.
// If the identifier is [token.BlankIdent], then this method is a no-op.
func (e *environment) Declare(tok token.Token) {
	if tok.Literal == token.BlankIdent {
		return
	}
	if _, ok := e.valuesByIdent[tok.Literal]; ok {
		panic(lox.NewErrorFromToken(tok, "%s has already been declared", tok.Literal))
	}
	e.valuesByIdent[tok.Literal] = nil
}

// Define declares an identifier in this environment and defines it with a value.
// If the identifier has already been declared in this environment, then a runtime error is raised.
// If the identifier is [token.BlankIdent], then this method is a no-op.
// This method should be used for defining values which originated from an assignment in code. For example, a variable
// or function declaration. Otherwise, use [*environment.Set].
func (e *environment) Define(tok token.Token, value loxObject) {
	if tok.Literal == token.BlankIdent {
		return
	}
	if value == nil {
		panic(fmt.Sprintf("attempt to define %s to nil", tok.Literal))
	}
	if _, ok := e.valuesByIdent[tok.Literal]; ok {
		panic(lox.NewErrorFromToken(tok, "%s has already been declared", tok.Literal))
	}
	e.valuesByIdent[tok.Literal] = value
}

// Set declares an identifier in this environment and defines it with a value.
// If the identifier has already been declared in this environment, then this method panics.
// If the identifier is [token.BlankIdent], then this method is a no-op.
// This method should be used for defining values which did not originate from an assignment in code. For example,
// defining built-in functions or function arguments. Otherwise, use [*environment.Define].
func (e *environment) Set(ident string, value loxObject) {
	if ident == token.BlankIdent {
		return
	}
	if value == nil {
		panic(fmt.Sprintf("attempt to set %s to nil", ident))
	}
	if _, ok := e.valuesByIdent[ident]; ok {
		// It's a bug if we end up here
		panic(fmt.Sprintf("%s has already been declared", ident))
	}
	e.valuesByIdent[ident] = value
}

// Assign assigns a value to an identifier in this environment.
// If the identifier has not been defined in this environment, then a runtime error is raised.
// If the identifier is [token.BlankIdent], then this method is a no-op.
func (e *environment) Assign(tok token.Token, value loxObject) {
	if tok.Literal == token.BlankIdent {
		return
	}
	if value == nil {
		panic(fmt.Sprintf("attempt to assign nil to %s", tok.Literal))
	}
	_, ok := e.valuesByIdent[tok.Literal]
	if !ok {
		panic(lox.NewErrorFromToken(tok, "%s has not been declared", tok.Literal))
	}
	e.valuesByIdent[tok.Literal] = value
}

// AssignAt assigns a value to a variable in the environment distance levels up the parent chain.
func (e *environment) AssignAt(distance int, tok token.Token, value loxObject) {
	e.ancestor(distance).Assign(tok, value)
}

// Get returns the value of an identifier in this environment.
// If the identifier has not been declared or defined in this environment, then a runtime error is raised.
func (e *environment) Get(tok token.Token) loxObject {
	value, ok := e.valuesByIdent[tok.Literal]
	if !ok {
		panic(lox.NewErrorFromToken(tok, "%s has not been declared", tok.Literal))
	}
	if value == nil {
		panic(lox.NewErrorFromToken(tok, "%s has not been defined", tok.Literal))
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
