// Package token defines Token which represents a lexical token of the Lox programming language.
package token

import "fmt"

//go:generate go run golang.org/x/tools/cmd/stringer -type Type -linecomment

// Type is the type of a lexical token of Lox code.
type Type uint8

// The list of all token types.
const (
	Illegal Type = iota // ILLEGAL
	Comment             // COMMENT
	EOF

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
	Bang         // !
	Question     // ?
	Colon        // :

	// Brackets
	LeftParen  // (
	RightParen // )
	LeftBrace  // {
	RightBrace // }
)

// Token is a lexical token of Lox code.
type Token struct {
	Position Position
	Type     Type
	Literal  string
}

func (t Token) String() string {
	if t.Literal != "" {
		return t.Literal
	}
	return t.Type.String()
}

// Position is a position in a file.
type Position struct {
	File         *File
	Line, Column int
}

func (p Position) String() string {
	var prefix string
	if p.File != nil && p.File.Name != "" {
		prefix = p.File.Name + ":"
	}
	return fmt.Sprintf("%s%d:%d", prefix, p.Line, p.Column)
}

// File is a simple representation of a file.
type File struct {
	Name        string
	contents    []byte
	lineOffsets []int
}

// NewFile returns a new File with the given contents.
func NewFile(name string, contents []byte) *File {
	f := &File{
		Name:     name,
		contents: contents,
	}
	f.lineOffsets = append(f.lineOffsets, 0)
	for i := 0; i < len(contents); i++ {
		if contents[i] == '\n' {
			f.lineOffsets = append(f.lineOffsets, i+1)
		}
	}
	return f
}

// Line returns the nth line of the file.
func (f *File) Line(n int) string {
	low := f.lineOffsets[n-1]
	high := len(f.contents)
	if n < len(f.lineOffsets) {
		high = f.lineOffsets[n] - 1 // -1 to exclude the newline
	}
	line := string(f.contents[low:high])
	if n == len(f.lineOffsets) {
		line += " " // We treat EOF as a space at the end of the file
	}
	return line
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
