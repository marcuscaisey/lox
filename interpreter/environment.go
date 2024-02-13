package interpreter

import (
	"fmt"

	"github.com/marcuscaisey/golox/token"
)

type environment struct {
	parent *environment
	values map[string]loxObject
}

func newEnvironment(parent *environment) *environment {
	return &environment{
		parent: parent,
		values: make(map[string]loxObject),
	}
}

// Define defines a variable in this environment and assigns it an initial value.
// If the variable is already defined in this environment then a runtime error is raised.
func (e *environment) Define(tok token.Token, value loxObject) {
	if _, ok := e.values[tok.Literal]; ok {
		panic(&runtimeError{
			tok: tok,
			msg: fmt.Sprintf("%s has already been defined", tok.Literal),
		})
	}
	e.values[tok.Literal] = value
}

// Assign assigns a value to a variable.
// If the variable has not been defined then a runtime error is raised.
func (e *environment) Assign(tok token.Token, value loxObject) {
	if e == nil {
		panic(&runtimeError{
			tok: tok,
			msg: fmt.Sprintf("%s has not been defined", tok.Literal),
		})
	}

	if _, ok := e.values[tok.Literal]; ok {
		e.values[tok.Literal] = value
		return
	}

	e.parent.Assign(tok, value)
}

// Get returns the value of a variable.
// If the variable has not been defined then a runtime error is raised.
func (e *environment) Get(tok token.Token) loxObject {
	if e == nil {
		panic(&runtimeError{
			tok: tok,
			msg: fmt.Sprintf("%s has not been defined", tok.Literal),
		})
	}

	if value, ok := e.values[tok.Literal]; ok {
		if value == nil {
			panic(&runtimeError{
				tok: tok,
				msg: fmt.Sprintf("%s has not been initialised", tok.Literal),
			})
		}
		return value
	}

	return e.parent.Get(tok)
}
