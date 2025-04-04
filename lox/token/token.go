// Package token declares the type representing a lexical token of Lox code.
package token

import (
	"cmp"
	"fmt"
	"unicode"
	"unicode/utf16"

	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/lox/lox/ansi"
)

func init() {
	for i := range typesEnd {
		t := Type(i)
		if _, ok := typeStrings[t]; !ok && unicode.IsUpper(rune(t.String()[0])) {
			panic(fmt.Sprintf("typeStrings is missing entry for Type %s", t.String()))
		}
	}
}

const (
	// PlaceholderIdent is the special identifier which can be used as a placeholder in declarations and assignments.
	PlaceholderIdent = "_"
	// CurrentInstanceIdent is the identifier used to refer the current instance of the class in a method.
	CurrentInstanceIdent = thisIdent
	// ConstructorIdent is the identifier used for the constructor method for classes.
	ConstructorIdent = "init"

	thisIdent = "this"
)

//go:generate go tool stringer -type Type

// Type is the type of a lexical token of Lox code.
type Type int

// The list of all token types.
const (
	Illegal Type = iota
	EOF

	// Keywords
	keywordsStart
	Print
	Var
	True
	False
	Nil
	If
	Else
	And
	Or
	While
	For
	Break
	Continue
	Fun
	Return
	Class
	This
	Super
	Static
	Get
	Set
	keywordsEnd

	// Literals
	Ident
	String
	Number
	Comment

	// Symbols
	Semicolon
	Comma
	Dot
	Equal
	Plus
	Minus
	Asterisk
	Slash
	Percent
	Less
	LessEqual
	Greater
	GreaterEqual
	EqualEqual
	BangEqual
	Bang
	Question
	Colon
	LeftParen
	RightParen
	LeftBrace
	RightBrace

	typesEnd
)

var typeStrings = map[Type]string{
	Illegal:       "illegal",
	EOF:           "EOF",
	keywordsStart: "keywordsStart",
	Print:         "print",
	Var:           "var",
	True:          "true",
	False:         "false",
	Nil:           "nil",
	If:            "if",
	Else:          "else",
	And:           "and",
	Or:            "or",
	While:         "while",
	For:           "for",
	Break:         "break",
	Continue:      "continue",
	Fun:           "fun",
	Return:        "return",
	Class:         "class",
	This:          thisIdent,
	Super:         "super",
	Static:        "static",
	Get:           "get",
	Set:           "set",
	typesEnd:      "typesEnd",
	Ident:         "identifier",
	String:        "string",
	Number:        "number",
	Comment:       "comment",
	Semicolon:     ";",
	Comma:         ",",
	Dot:           ".",
	Equal:         "=",
	Plus:          "+",
	Minus:         "-",
	Asterisk:      "*",
	Slash:         "/",
	Percent:       "%",
	Less:          "<",
	LessEqual:     "<=",
	Greater:       ">",
	GreaterEqual:  ">=",
	EqualEqual:    "==",
	BangEqual:     "!=",
	Bang:          "!",
	Question:      "?",
	Colon:         ":",
	LeftParen:     "(",
	RightParen:    ")",
	LeftBrace:     "{",
	RightBrace:    "}",
	keywordsEnd:   "keywordsEnd",
}

var keywordTypesByIdent = func() map[string]Type {
	keywordTypesByIdent := make(map[string]Type, keywordsEnd-keywordsStart)
	for i := keywordsStart; i < keywordsEnd; i++ {
		keywordTypesByIdent[typeStrings[i]] = i
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
		fmt.Fprintf(f, "'%s'", typeStrings[t])
	case 's':
		fmt.Fprint(f, t.String())
	default:
		fmt.Fprintf(f, fmt.FormatString(f, verb), uint8(t))
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

func (t Token) String() string {
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
//	-1 if p is comes before other in the file,
//	 0 if p and other are the same position,
//	+1 if p comes after other in the file.
func (p Position) Compare(other Position) int {
	if p.Line == other.Line {
		return cmp.Compare(p.Column, other.Column)
	}
	return cmp.Compare(p.Line, other.Line)
}

// ColumnUTF16 returns the column offset in UTF-16 code units.
func (p Position) ColumnUTF16() int {
	line := p.File.Line(p.Line)
	return len(utf16.Encode([]rune(string(line[:p.Column]))))
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

// Format implements fmt.Formatter. All verbs have the default behaviour, except for 'm' (message) which formats the
// position for use in an error message.
func (p Position) Format(f fmt.State, verb rune) {
	switch verb {
	case 'm':
		var prefix string
		if p.File != nil && p.File.Name != "" {
			prefix = ansi.Sprint("${CYAN}", p.File.Name, "${DEFAULT}:")
		}
		line := p.File.Line(p.Line)
		col := runewidth.StringWidth(string(line[:p.Column])) + 1
		ansi.Fprint(f, prefix, "${YELLOW}", p.Line, "${DEFAULT}:${YELLOW}", col, "${DEFAULT}")
	case 's':
		fmt.Fprint(f, p.String())
	default:
		fmt.Fprintf(f, fmt.FormatString(f, verb), p)
	}
}

// Range describes a range of characters in the source code.
type Range interface {
	Start() Position // Start returns the position of the first character of the range.
	End() Position   // End returns the position of the character immediately after the range.
}

// Ranges is a slice of [Range].
type Ranges[T Range] []T

// Start returns the position of the first character of the first range.
func (r Ranges[T]) Start() Position {
	return r[0].Start()
}

// End returns the position of the character immediately after the last range.
func (r Ranges[T]) End() Position {
	return r[len(r)-1].End()
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
