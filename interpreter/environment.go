package interpreter

import (
	"fmt"

	"github.com/marcuscaisey/golox/token"
)

// environment contains the values of all variables in scope.
type environment struct {
	values map[string]loxObject
}

func newEnvironment() *environment {
	return &environment{
		values: make(map[string]loxObject),
	}
}

// Define defines a variable and assigns it an initial value.
// If the variable is already defined then a runtime error is raised.
func (e *environment) Define(tok token.Token, value loxObject) {
	if _, ok := e.values[tok.Literal]; ok {
		panic(&runtimeError{
			tok: tok,
			msg: fmt.Sprintf("%s has already been defined", tok.Literal),
		})
	}
	e.values[tok.Literal] = value
}

// Assign assigns a new value to a variable.
// If the variable has not been defined then a runtime error is raised.
func (e *environment) Assign(tok token.Token, value loxObject) {
	if _, ok := e.values[tok.Literal]; !ok {
		panic(&runtimeError{
			tok: tok,
			msg: fmt.Sprintf("%s has not been defined", tok.Literal),
		})
	}
	e.values[tok.Literal] = value
}

// Get returns the value of a variable.
// If the variable has not been defined then a runtime error is raised.
func (e *environment) Get(tok token.Token) loxObject {
	value, ok := e.values[tok.Literal]
	if !ok {
		panic(&runtimeError{
			tok: tok,
			msg: fmt.Sprintf("%s has not been defined", tok.Literal),
		})
	}
	return value
}
