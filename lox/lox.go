// Package lox implements functionality used by most packages.
package lox

const (
	// BuiltinClock is the name of the built-in clock function.
	BuiltinClock string = "clock"
	// BuiltinType is the name of the built-in type function.
	BuiltinType string = "type"
	// BuiltinError is the name of the built-in error function.
	BuiltinError string = "error"
)

// AllBuiltins contains the names of all objects that are built-in to the language.
var AllBuiltins = []string{BuiltinClock, BuiltinType, BuiltinError}
