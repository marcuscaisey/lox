// Package token defines Token which represents a lexical token of the Lox programming language.
package token

//go:generate go run golang.org/x/tools/cmd/stringer -type Type -linecomment

// Type is the type of a lexical token of Lox code.
type Type uint8

// The list of all token types.
const (
	unknown Type = iota

	// Keywords
	keywordsStart
	Print    // print
	Var      // var
	True     // true
	False    // false
	Nil      // nil
	If       // if
	Else     // else
	And      // and
	Or       // or
	While    // while
	For      // for
	Function // fun
	Return   // return
	Class    // class
	This     // this
	Super    // super
	keywordsEnd

	// Delimiters
	Semicolon // ;
	Comma     // ,
	Dot       // .

	// Literals
	Ident  // identifier
	String // string
	Number // number

	// Operators
	Assign       // =
	Plus         // +
	Minus        // -
	Asterisk     // *
	Slash        // /
	Less         // <
	LessEqual    // <=
	Greater      // >
	GreaterEqual // >=
	Equal        // ==
	NotEqual     // !=
	Not          // !

	// Brackets
	OpenParen  // (
	CloseParen // )
	OpenBrace  // {
	CloseBrace // }

	EOF
)

// Token is a lexical token of Lox code.
type Token struct {
	Type    Type
	Lexeme  string
	Literal any
	Line    int
	Byte    int
}

var keywordTypesByIdent = func() map[string]Type {
	m := make(map[string]Type, keywordsEnd-keywordsStart-2)
	for i := keywordsStart + 1; i < keywordsEnd; i++ {
		m[Type(i).String()] = Type(i)
	}
	return m
}()

// LookupIdent returns the keyword Type associated with the given ident if its a keyword, otherwise Ident.
func LookupIdent(ident string) Type {
	if keywordType, ok := keywordTypesByIdent[ident]; ok {
		return keywordType
	}
	return Ident
}
