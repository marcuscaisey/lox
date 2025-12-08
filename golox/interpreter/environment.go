package interpreter

import (
	"fmt"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerr"
)

// environment stores the values of identifiers in a lexical scope.
type environment interface {
	// Child create a new child of this environment.
	// Identifiers in the parent environment are visible in the child.
	// Identifiers declared in the child environment are not visible in the parent.
	// Identifiers declared in the child environment shadow identifiers with the same name in the parent environment.
	Child() environment
	// Declare declares an identifier and returns the updated environment.
	// This should be used for identifiers that originate from a declaration in code, like a variable declaration.
	Declare(ident *ast.Ident) environment
	// Define defines an identifier and returns the updated environment.
	// This should be used for identifiers that don't originate from a declaration in code, like a function parameter.
	Define(name string, value loxObject) environment
	// Assign assigns a value to an identifier and returns the updated environment.
	Assign(ident *ast.Ident, value loxObject)
	// Get returns the value of the identifier.
	Get(ident *ast.Ident) loxObject
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

func (e *globalEnvironment) Declare(ident *ast.Ident) environment {
	e.values[ident.String()] = loxNil{}
	return e
}

func (e *globalEnvironment) Define(name string, value loxObject) environment {
	if value == nil {
		panic(fmt.Sprintf("attempt to set %s to nil", name))
	}
	if _, ok := e.values[name]; !ok {
		e.values[name] = value
		return e
	} else {
		panic(fmt.Sprintf("%s has already been declared", name))
	}
}

func (e *globalEnvironment) Assign(ident *ast.Ident, value loxObject) {
	if value == nil {
		panic(fmt.Sprintf("attempt to assign nil to %s", ident.String()))
	}
	if _, ok := e.values[ident.String()]; ok {
		e.values[ident.String()] = value
	} else {
		panic(loxerr.Newf(ident, loxerr.Fatal, "%s has not been declared", ident.String()))
	}
}

func (e *globalEnvironment) Get(ident *ast.Ident) loxObject {
	if value, ok := e.values[ident.String()]; ok {
		return value
	} else {
		panic(loxerr.Newf(ident, loxerr.Fatal, "%s has not been declared", ident.String()))
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

func (e *localEnvironment) Declare(ident *ast.Ident) environment {
	return newLocalEnvironment(e, ident.String(), loxNil{})
}

func (e *localEnvironment) Define(name string, value loxObject) environment {
	if value == nil {
		panic(fmt.Sprintf("attempt to set %s to nil", name))
	}
	return newLocalEnvironment(e, name, value)
}

func (e *localEnvironment) Assign(ident *ast.Ident, value loxObject) {
	if value == nil {
		panic(fmt.Sprintf("attempt to assign nil to %s", ident.String()))
	}
	if ident.String() == e.name {
		e.value = value
	} else if e.parent != nil {
		e.parent.Assign(ident, value)
	} else {
		panic(loxerr.Newf(ident, loxerr.Fatal, "%s has not been declared", ident.String()))
	}
}

func (e *localEnvironment) Get(ident *ast.Ident) loxObject {
	if ident.String() == e.name {
		return e.value
	} else if e.parent != nil {
		return e.parent.Get(ident)
	} else {
		panic(loxerr.Newf(ident, loxerr.Fatal, "%s has not been declared", ident.String()))
	}
}
