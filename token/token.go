// Package token defines Token which represents a lexical token of the Lox programming language.
package token

import "fmt"

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
	literalsStart
	Ident  // identifier
	String // string
	Number // number
	literalsEnd

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
	Bang         // !

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
	Pos     Position
}

func (t Token) String() string {
	if t.isLiteral() {
		return t.Lexeme
	}
	return t.Type.String()
}

func (t Token) isLiteral() bool {
	return literalsStart < t.Type && t.Type < literalsEnd
}

// Position is a position in a file.
type Position struct {
	Line, Byte int
}

func (p Position) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Byte)
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
