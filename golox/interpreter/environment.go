package interpreter

import (
	"fmt"
	"slices"
	"strings"

	"github.com/marcuscaisey/lox/lox"
	"github.com/marcuscaisey/lox/lox/token"
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

func (e *environment) String() string {
	_, s := e.string()
	return s
}

func (e *environment) string() (prefix string, s string) {
	var b strings.Builder
	firstLinePrefix := ""
	if e.parent != nil {
		parentPrefix, parentString := e.parent.string()
		fmt.Fprintf(&b, "%s\n", parentString)
		prefix = parentPrefix + "   "
		firstLinePrefix = parentPrefix + "└──"
	}

	if len(e.valuesByIdent) == 0 {
		fmt.Fprintf(&b, "%s<empty>", firstLinePrefix)
		return prefix, b.String()
	}

	idents := make([]string, 0, len(e.valuesByIdent))
	for ident := range e.valuesByIdent {
		idents = append(idents, ident)
	}
	slices.Sort(idents)
	for i, ident := range idents {
		prefix := prefix
		if i == 0 {
			prefix = firstLinePrefix
		}
		fmt.Fprintf(&b, "%s%s: %s\n", prefix, ident, e.valuesByIdent[ident])
	}
	return prefix, strings.TrimSuffix(b.String(), "\n")
}

// Child creates a new child environment of this environment.
func (e *environment) Child() *environment {
	env := newEnvironment()
	env.parent = e
	return env
}

// Declare declares an identifier in this environment.
// If the identifier has already been declared in this environment, then an error is raised.
// If the identifier is [token.PlaceholderIdent], then this method is a no-op.
func (e *environment) Declare(tok token.Token) {
	if tok.Lexeme == token.PlaceholderIdent {
		return
	}
	if _, ok := e.valuesByIdent[tok.Lexeme]; ok {
		panic(lox.NewErrorf(tok, "%s has already been declared", tok.Lexeme))
	}
	e.valuesByIdent[tok.Lexeme] = nil
}

// Define declares an identifier in this environment and defines it with a value.
// If the identifier has already been declared in this environment, then an error is raised.
// If the identifier is [token.PlaceholderIdent], then this method is a no-op.
// This method should be used for defining values which originated from an assignment in code. For example, a variable
// or function declaration. Otherwise, use [*environment.Set].
func (e *environment) Define(tok token.Token, value loxObject) {
	if tok.Lexeme == token.PlaceholderIdent {
		return
	}
	if value == nil {
		panic(fmt.Sprintf("attempt to define %s to nil", tok.Lexeme))
	}
	if _, ok := e.valuesByIdent[tok.Lexeme]; ok {
		panic(lox.NewErrorf(tok, "%s has already been declared", tok.Lexeme))
	}
	e.valuesByIdent[tok.Lexeme] = value
}

// Set declares an identifier in this environment and defines it with a value.
// If the identifier has already been declared in this environment, then this method panics.
// If the identifier is [token.PlaceholderIdent], then this method is a no-op.
// This method should be used for defining values which did not originate from an assignment in code. For example,
// defining built-in functions or function arguments. Otherwise, use [*environment.Define].
func (e *environment) Set(ident string, value loxObject) {
	if ident == token.PlaceholderIdent {
		return
	}
	if value == nil {
		// It's a bug if we end up here
		panic(fmt.Sprintf("attempt to set %s to nil", ident))
	}
	if _, ok := e.valuesByIdent[ident]; ok {
		// It's a bug if we end up here
		panic(fmt.Sprintf("%s has already been declared", ident))
	}
	e.valuesByIdent[ident] = value
}

// Assign assigns a value to an identifier in this environment.
// If the identifier has not been defined in this environment, then an error is raised.
// If the identifier is [token.PlaceholderIdent], then this method is a no-op.
func (e *environment) Assign(tok token.Token, value loxObject) {
	if tok.Lexeme == token.PlaceholderIdent {
		return
	}
	if value == nil {
		panic(fmt.Sprintf("attempt to assign nil to %s", tok.Lexeme))
	}
	_, ok := e.valuesByIdent[tok.Lexeme]
	if !ok {
		panic(lox.NewErrorf(tok, "%s has not been declared", tok.Lexeme))
	}
	e.valuesByIdent[tok.Lexeme] = value
}

// AssignAt assigns a value to a variable in the environment distance levels up the parent chain.
func (e *environment) AssignAt(distance int, tok token.Token, value loxObject) {
	e.ancestor(distance).Assign(tok, value)
}

// Get returns the value of an identifier in this environment.
// If the identifier has not been declared or defined in this environment, then an error is raised.
func (e *environment) Get(tok token.Token) loxObject {
	value, ok := e.valuesByIdent[tok.Lexeme]
	if !ok {
		panic(lox.NewErrorf(tok, "%s has not been declared", tok.Lexeme))
	}
	if value == nil {
		panic(lox.NewErrorf(tok, "%s has not been defined", tok.Lexeme))
	}
	return value
}

// Get returns the value of an identifier in this environment.
// If the identifier has not been declared or defined in this environment, then this method panics.
// This method should be used for accesses which did not originate from an expression in code. Otherwise, use
// [*environment.Get].
func (e *environment) GetByIdent(ident string) loxObject {
	value, ok := e.valuesByIdent[ident]
	if !ok {
		// It's a bug if we end up here
		panic(fmt.Sprintf("%s has not been declared", ident))
	}
	if value == nil {
		// It's a bug if we end up here
		panic(fmt.Sprintf("%s has not been defined", ident))
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
