// Package token declares the type representing a lexical token of Lox code.
package token

import (
	"cmp"
	"fmt"

	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/lox/golox/ansi"
)

// Constants for special identifiers.
const (
	IdentBlank = "_"
	IdentInit  = "init"
)

//go:generate go tool stringer -type Type -linecomment

// Type is the type of a lexical token of Lox code.
type Type int

// The list of all token types.
const (
	Illegal Type = iota
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
	Break    // break
	Continue // continue
	Fun      // fun
	Return   // return
	Class    // class
	This     // this
	Super    // super
	Static   // static
	Get      // get
	Set      // set
	keywordsEnd

	// Literals
	Ident
	String
	Number
	Comment

	// Symbols
	symbolsStart
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
	LeftBrack    // [
	RightBrack   // ]
	LeftBrace    // {
	RightBrace   // }
	symbolsEnd

	typesEnd
)

var keywordTypesByIdent = func() map[string]Type {
	keywordTypesByIdent := make(map[string]Type, keywordsEnd-keywordsStart)
	for i := keywordsStart; i < keywordsEnd; i++ {
		keywordTypesByIdent[i.String()] = i
	}
	return keywordTypesByIdent
}()

// IdentType returns the type of the keyword with the given identifier, or Ident if the identifier is not a
// keyword.
func IdentType(ident string) Type {
	if keywordType, ok := keywordTypesByIdent[ident]; ok {
		return keywordType
	}
	return Ident
}

// Format implements fmt.Formatter. All verbs have the default behaviour, except for 'm' (message) which formats the
// type for use in an error message.
func (t Type) Format(f fmt.State, verb rune) {
	switch verb {
	case 'm':
		fmt.Fprintf(f, "'%s'", t.String())
	default:
		fmt.Fprint(f, t.String())
	}
}

// Token is a lexical token of Lox code.
type Token struct {
	StartPos Position // Position of the first character of the token
	EndPos   Position // Position of the character immediately after the token
	Type     Type
	Lexeme   string
}

// Start returns the position of the first character of the token.
func (t Token) Start() Position {
	return t.StartPos
}

// End returns the position of the character immediately after the token.
func (t Token) End() Position {
	return t.EndPos
}

// IsZero reports whether t is the zero value.
func (t Token) IsZero() bool {
	return t == Token{}
}

func (t Token) String() string {
	if t.Type == EOF {
		return fmt.Sprintf("%s: [%s]", t.StartPos, t.Type)
	}
	if (keywordsStart < t.Type && t.Type < keywordsEnd) || (symbolsStart < t.Type && t.Type < symbolsEnd) {
		return fmt.Sprintf("%s: %s", t.StartPos, t.Lexeme)
	}
	return fmt.Sprintf("%s: %s [%s]", t.StartPos, t.Lexeme, t.Type)
}

// Position is a position in a file.
type Position struct {
	File   *File
	Line   int // 1-based line number
	Column int // 0-based byte offset from the start of the line
}

// Compare returns
//
//	-1 if p's file comes before other's or p and other are in the same file and p comes before other,
//	 0 if p and other are the same position in the same file,
//	+1 if p's file comes after other's or p and other are in the same file and p comes after other.
func (p Position) Compare(other Position) int {
	if p.File.Name != other.File.Name {
		return cmp.Compare(p.File.Name, other.File.Name)
	}
	if p.Line == other.Line {
		return cmp.Compare(p.Column, other.Column)
	}
	return cmp.Compare(p.Line, other.Line)
}

func (p Position) String() string {
	line := p.File.Line(p.Line)
	col := runewidth.StringWidth(string(line[:p.Column])) + 1
	return fmt.Sprintf("%d:%d", p.Line, col)
}

// Format implements fmt.Formatter. All verbs have the default behaviour, except for 'm' (message) which formats the
// position for use in an error message.
func (p Position) Format(f fmt.State, verb rune) {
	switch verb {
	case 'm':
		line := p.File.Line(p.Line)
		col := runewidth.StringWidth(string(line[:p.Column])) + 1
		ansi.Fprint(f, "${YELLOW}", p.Line, "${DEFAULT}:${YELLOW}", col, "${DEFAULT}")
	case 's':
		fmt.Fprint(f, p.String())
	default:
		fmt.Fprintf(f, fmt.FormatString(f, verb), struct {
			File   *File
			Line   int
			Column int
		}(p))
	}
}

// Range describes a range of characters in the source code.
type Range interface {
	Start() Position // Start returns the position of the first character of the range.
	End() Position   // End returns the position of the character immediately after the range.
}

// File is a simple representation of a file.
type File struct {
	Name        string
	Contents    []byte
	lineOffsets []int
}

// NewFile returns a new File with the given contents.
func NewFile(name string, contents []byte) *File {
	f := &File{
		Name:     name,
		Contents: contents,
	}
	f.lineOffsets = append(f.lineOffsets, 0)
	for i := range contents {
		if contents[i] == '\n' {
			f.lineOffsets = append(f.lineOffsets, i+1)
		}
	}
	return f
}

// Line returns the nth (1-based) line of the file.
func (f *File) Line(n int) []byte {
	low := f.lineOffsets[n-1]
	high := len(f.Contents)
	if n < len(f.lineOffsets) {
		high = f.lineOffsets[n] - 1 // -1 to exclude the newline
	}
	line := f.Contents[low:high]
	return line
}
