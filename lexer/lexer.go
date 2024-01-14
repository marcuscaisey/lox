// Package lexer implements a lexer for Lox source code.
package lexer

import (
	"fmt"
	"io"
	"strings"

	"github.com/marcuscaisey/golox/token"
)

const eof = -1

// ErrorHandler is the function which handles syntax errors encountered during lexing.
// It's passed the offending token and a message describing the error.
type ErrorHandler func(tok token.Token, msg string)

// Lexer converts Lox source code into lexical tokens.
// Tokens are read from the lexer using the Next method.
// Syntax errors are handled by calling the error handler function which can be set using SetErrorHandler. The default
// error handler is a no-op.
type Lexer struct {
	// Immutable state
	src        []byte
	errHandler ErrorHandler

	// Mutable state
	ch         rune           // character currently being considered
	pos        token.Position // position of character currently being considered
	readOffset int            // position of next character to be read
}

// New constructs a Lexer which will lex the source code read from an io.Reader.
func New(r io.Reader) (*Lexer, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("constructing lexer: %s", err)
	}
	errHandler := func(token.Token, string) {}
	filename := name(r)

	l := &Lexer{
		src:        src,
		errHandler: errHandler,
		pos: token.Position{
			File: token.NewFile(filename, src),
			Line: 1,
			// BUG: This is not correct for multi-byte (e.g. UTF-8) characters.
			Column: 0,
		},
	}

	l.next()

	return l, nil
}

func name(v any) string {
	if n, ok := v.(interface{ Name() string }); ok {
		return n.Name()
	}
	return ""
}

// SetErrorHandler sets the error handler function which will be called when a syntax error is encountered.
func (l *Lexer) SetErrorHandler(errHandler ErrorHandler) {
	l.errHandler = errHandler
}

// Next returns the next token. An EOF token is returned if the end of the source code has been reached.
func (l *Lexer) Next() token.Token {
	l.skipWhitespace()

	tok := token.Token{Position: l.pos}

	switch {
	case l.ch == eof:
		tok.Type = token.EOF
		// Treating EOF as a space instead of an empty string makes it easier to display (e.g. in syntax errors) without
		// having to special case it.
		tok.Literal = " "
	case l.ch == ';':
		tok.Type = token.Semicolon
	case l.ch == ',':
		tok.Type = token.Comma
	case l.ch == '.':
		tok.Type = token.Dot
	case l.ch == '=':
		tok.Type = token.Assign
		if l.peek() == '=' {
			l.next()
			tok.Type = token.Equal
		}
	case l.ch == '+':
		tok.Type = token.Plus
	case l.ch == '-':
		tok.Type = token.Minus
	case l.ch == '*':
		tok.Type = token.Asterisk
	case l.ch == '/':
		tok.Type = token.Slash
		if l.peek() == '/' {
			l.next()
			l.next()
			l.skipSingleLineComment()
			return l.Next()
		}
		if l.peek() == '*' {
			l.next()
			l.next()
			if comment, terminated := l.consumeMultiLineComment(); !terminated {
				tok.Type = token.Comment
				tok.Literal = comment
				l.errHandler(tok, "unterminated multi-line comment")
			}
			return l.Next()
		}
	case l.ch == '<':
		tok.Type = token.Less
		if l.peek() == '=' {
			l.next()
			tok.Type = token.LessEqual
		}
	case l.ch == '>':
		tok.Type = token.Greater
		if l.peek() == '=' {
			l.next()
			tok.Type = token.GreaterEqual
		}
	case l.ch == '!':
		tok.Type = token.Bang
		if l.peek() == '=' {
			l.next()
			tok.Type = token.NotEqual
		}
	case l.ch == '?':
		tok.Type = token.Question
	case l.ch == ':':
		tok.Type = token.Colon
	case l.ch == '(':
		tok.Type = token.LeftParen
	case l.ch == ')':
		tok.Type = token.RightParen
	case l.ch == '{':
		tok.Type = token.LeftBrace
	case l.ch == '}':
		tok.Type = token.RightBrace
	case l.ch == '"':
		tok.Type = token.String
		lit, terminated := l.consumeString()
		tok.Literal = lit
		if !terminated {
			tok.Type = token.Illegal
			l.errHandler(tok, "unterminated string literal")
		}
		return tok
	case isDigit(l.ch):
		tok.Type = token.Number
		tok.Literal = l.consumeNumber()
		return tok
	case isAlpha(l.ch):
		ident := l.consumeIdent()
		tok.Type = token.LookupIdent(ident)
		if tok.Type == token.Ident {
			tok.Literal = ident
		}
		return tok
	default:
		tok.Type = token.Illegal
		tok.Literal = string(l.ch)
		l.errHandler(tok, fmt.Sprintf("illegal character %#U", l.ch))
	}
	l.next()

	return tok
}

func (l *Lexer) skipWhitespace() {
	for isWhitespace(l.ch) {
		l.next()
	}
}

func (l *Lexer) skipSingleLineComment() {
	for l.ch != '\n' && l.ch != eof {
		l.next()
	}
}

func (l *Lexer) consumeMultiLineComment() (comment string, terminated bool) {
	var b strings.Builder
	b.WriteString("/*")
	// Multi-line comments can be nested
	openComments := 1 // There's already a comment open when this method is called
	for openComments > 0 && l.ch != eof {
		b.WriteRune(l.ch)
		if l.ch == '/' && l.peek() == '*' {
			l.next()
			openComments++
		} else if l.ch == '*' && l.peek() == '/' {
			l.next()
			openComments--
		}
		l.next()
	}
	return b.String(), openComments == 0
}

func (l *Lexer) consumeNumber() string {
	var b strings.Builder
	for isDigit(l.ch) {
		b.WriteRune(l.ch)
		l.next()
	}
	if l.ch == '.' && isDigit(l.peek()) {
		b.WriteRune(l.ch)
		l.next()
		b.WriteRune(l.ch)
		l.next()
		for isDigit(l.ch) {
			b.WriteRune(l.ch)
			l.next()
		}
	}
	return b.String()
}

func (l *Lexer) consumeString() (s string, terminated bool) {
	l.next()
	var b strings.Builder
	b.WriteRune('"')
	for {
		if l.ch == eof || l.ch == '\n' || l.ch == '\r' {
			return b.String(), false
		}
		ch := l.ch
		b.WriteRune(ch)
		l.next()
		if ch == '"' {
			return b.String(), true
		}
	}
}

func (l *Lexer) consumeIdent() string {
	var b strings.Builder
	for isAlphaNumeric(l.ch) {
		b.WriteRune(l.ch)
		l.next()
	}
	return b.String()
}

func isWhitespace(r rune) bool {
	switch r {
	case ' ', '\r', '\t', '\n':
		return true
	default:
		return false
	}
}

func isDigit(r rune) bool {
	return '0' <= r && r <= '9'
}

func isAlpha(r rune) bool {
	return ('a' <= r && r <= 'z') || ('A' <= r && r <= 'Z') || r == '_'
}

func isAlphaNumeric(r rune) bool {
	return isAlpha(r) || isDigit(r)
}

// next reads the next character into s.ch and advances the lexer.
// If the end of the source code has been reached, s.ch is set to eof.
func (l *Lexer) next() {
	if l.ch == eof {
		return
	}

	if l.ch == '\n' {
		l.pos.Line++
		l.pos.Column = 1
	} else {
		l.pos.Column++
	}

	if l.readOffset == len(l.src) {
		l.ch = eof
		return
	}

	l.ch = rune(l.src[l.readOffset])
	l.readOffset++
}

// peek returns the next character without advancing the lexer.
// If the end of the source code has been reached, eof is returned.
func (l *Lexer) peek() rune {
	if l.readOffset >= len(l.src) {
		return eof
	}
	return rune(l.src[l.readOffset])
}
