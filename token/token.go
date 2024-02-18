// Package token defines Token which represents a lexical token of the Lox programming language.
package token

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/mattn/go-runewidth"
)

//go:generate go run golang.org/x/tools/cmd/stringer -type Type -linecomment

// Type is the type of a lexical token of Lox code.
type Type uint8

// The list of all token types.
const (
	Illegal Type = iota // ILLEGAL
	EOF                 // EOF

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

	// Literals
	Ident  // identifier
	String // string
	Number // number

	// Symbols
	Semicolon    // ;
	Comma        // ,
	Dot          // .
	Equal        // =
	Plus         // +
	Minus        // -
	Asterisk     // *
	Slash        // /
	Percent      // %
	Less         // <
	LessEqual    // <=
	Greater      // >
	GreaterEqual // >=
	EqualEqual   // ==
	BangEqual    // !=
	Bang         // !
	Question     // ?
	Colon        // :
	LeftParen    // (
	RightParen   // )
	LeftBrace    // {
	RightBrace   // }
)

// Format implements fmt.Formatter. All verbs have the default behaviour, except for 'h' (highlight) which prints the
// type in cyan.
func (t Type) Format(f fmt.State, verb rune) {
	switch verb {
	case 'h':
		fmt.Fprint(f, color.CyanString(t.String()))
	case 's', 'q', 'v', 'x', 'X':
		if !f.Flag('#') {
			fmt.Fprintf(f, fmt.FormatString(f, verb), t.String())
			break
		}
		fallthrough
	default:
		fmt.Fprintf(f, fmt.FormatString(f, verb), uint8(t))
	}
}

// Token is a lexical token of Lox code.
type Token struct {
	Start   Position // Position of the first character of the token
	End     Position // Position of the character immediately after the token
	Type    Type
	Literal string
}

func (t Token) String() string {
	if t.Literal != "" {
		return t.Literal
	}
	return t.Type.String()
}

// Position is a position in a file.
type Position struct {
	File   *File
	Line   int // 1-based line number
	Column int // 0-based byte offset from the start of the line
}

func (p Position) String() string {
	var prefix string
	if p.File != nil && p.File.Name != "" {
		prefix = p.File.Name + ":"
	}
	line := p.File.Line(p.Line)
	col := runewidth.StringWidth(string(line[:p.Column])) + 1
	return fmt.Sprintf("%s%d:%d", prefix, p.Line, col)
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
func (f *File) Line(n int) []byte {
	low := f.lineOffsets[n-1]
	high := len(f.contents)
	if n < len(f.lineOffsets) {
		high = f.lineOffsets[n] - 1 // -1 to exclude the newline
	}
	line := f.contents[low:high]
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
