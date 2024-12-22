package interpreter

import (
	"fmt"

	"github.com/marcuscaisey/lox/lox"
	"github.com/marcuscaisey/lox/lox/token"
)

// environment stores the values of variables in a lexical scope.
type environment interface {
	// Child create a new child of this environment.
	// Variables in the parent environment are visible in the child.
	// Variables declared in the child environment are not visible in the parent.
	// Variables declared in the child environment shadow variables with the same name in the parent environment.
	Child() environment
	// Declare declares a variable and returns the updated environment.
	Declare(name string) environment
	// Define declares a variable, assigns a value to it, and returns the updated environment.
	Define(name string, value loxObject) environment
	// Assign assigns a value to the variable identified by ident and returns the updated environment.
	Assign(ident token.Token, value loxObject)
	// Get returns the value of the variable identified by ident.
	Get(ident token.Token) loxObject
}

// globalEnvironment is the environment for the global scope.
type globalEnvironment struct {
	values map[string]loxObject
}

func newGlobalEnvironment() *globalEnvironment {
	return &globalEnvironment{
		values: map[string]loxObject{},
	}
}

func (e *globalEnvironment) Child() environment {
	return newLocalEnvironment(e, "", nil)
}

func (e *globalEnvironment) Declare(name string) environment {
	if _, ok := e.values[name]; !ok {
		e.values[name] = nil
		return e
	} else {
		// This should have been caught by [analysis.ResolveIdents].
		panic(fmt.Sprintf("%s has already been declared", name))
	}
}

func (e *globalEnvironment) Define(name string, value loxObject) environment {
	if value == nil {
		panic(fmt.Sprintf("attempt to set %s to nil", name))
	}
	e.Declare(name)
	e.values[name] = value
	return e
}

func (e *globalEnvironment) Assign(ident token.Token, value loxObject) {
	if value == nil {
		panic(fmt.Sprintf("attempt to assign nil to %s", ident.Lexeme))
	}
	if _, ok := e.values[ident.Lexeme]; ok {
		e.values[ident.Lexeme] = value
	} else {
		panic(lox.NewErrorf(ident, "%s has not been declared", ident.Lexeme))
	}
}

func (e *globalEnvironment) Get(ident token.Token) loxObject {
	if value, ok := e.values[ident.Lexeme]; ok {
		if value != nil {
			return value
		} else {
			panic(lox.NewErrorf(ident, "%s has not been defined", ident.Lexeme))
		}
	} else {
		panic(lox.NewErrorf(ident, "%s has not been declared", ident.Lexeme))
	}
}

// localEnvironment is the environment for a local scope.
type localEnvironment struct {
	parent environment
	name   string
	value  loxObject
}

func newLocalEnvironment(parent environment, name string, value loxObject) *localEnvironment {
	return &localEnvironment{
		parent: parent,
		name:   name,
		value:  value,
	}
}

func (e *localEnvironment) Child() environment {
	return e
}

func (e *localEnvironment) Declare(name string) environment {
	return newLocalEnvironment(e, name, nil)
}

func (e *localEnvironment) Define(name string, value loxObject) environment {
	if value == nil {
		panic(fmt.Sprintf("attempt to set %s to nil", name))
	}
	return newLocalEnvironment(e, name, value)
}

func (e *localEnvironment) Assign(ident token.Token, value loxObject) {
	if value == nil {
		panic(fmt.Sprintf("attempt to assign nil to %s", ident.Lexeme))
	}
	if ident.Lexeme == e.name {
		e.value = value
	} else if e.parent != nil {
		e.parent.Assign(ident, value)
	} else {
		// This should have been caught by [analysis.ResolveIdents].
		panic(fmt.Sprintf("%s has not been declared", ident.Lexeme))
	}
}

func (e *localEnvironment) Get(ident token.Token) loxObject {
	if ident.Lexeme == e.name {
		if e.value != nil {
			return e.value
		} else {
			// This should have been caught by [analysis.ResolveIdents].
			panic(fmt.Sprintf("%s has not been defined", ident.Lexeme))
		}
	} else if e.parent != nil {
		return e.parent.Get(ident)
	} else {
		// This should have been caught by [analysis.ResolveIdents].
		panic(fmt.Sprintf("%s has not been declared", ident.Lexeme))
	}
}
