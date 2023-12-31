// Package lexer implements a lexer for Lox source code.
package lexer

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/marcuscaisey/golox/token"
)

const eof = -1

// ErrorHandler is the function which handles syntax errors encountered during lexing.
// It's passed the position of the offending token and a message describing the error.
type ErrorHandler func(pos token.Position, msg string)

// Lexer converts Lox source code into lexical tokens.
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
// If errHandler is nil, it will be set to a no-op function.
func New(r io.Reader, errHandler ErrorHandler) (*Lexer, error) {
	if errHandler == nil {
		errHandler = func(pos token.Position, msg string) {}
	}
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("constructing lexer: reading source code: %s", err)
	}
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

// Lex converts the source code into a sequences of tokens.
func (l *Lexer) Lex() ([]token.Token, error) {
	var errs []error
	l.errHandler = func(pos token.Position, msg string) {
		errs = append(errs, fmt.Errorf("%s: syntax error: %s", pos, msg))
	}
	var tokens []token.Token
	for {
		tok := l.Next()
		tokens = append(tokens, tok)
		if tok.Type == token.EOF {
			break
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return tokens, nil
}

// Next returns the next token. An EOF token is returned if the end of the source code has been reached.
func (l *Lexer) Next() token.Token {
	l.skipWhitespace()

	tok := token.Token{Position: l.pos}

	if isDigit(l.ch) {
		tok.Type = token.Number
		tok.Literal = l.consumeNumber()
		return tok
	}

	if isAlpha(l.ch) {
		tok.Literal = l.consumeIdent()
		tok.Type = token.LookupIdent(tok.Literal)
		return tok
	}

	switch l.ch {
	case eof:
		tok.Type = token.EOF
	case ';':
		tok.Type = token.Semicolon
	case ',':
		tok.Type = token.Comma
	case '.':
		tok.Type = token.Dot
	case '=':
		tok.Type = token.Assign
		if l.peek() == '=' {
			l.next()
			tok.Type = token.Equal
		}
	case '+':
		tok.Type = token.Plus
	case '-':
		tok.Type = token.Minus
	case '*':
		tok.Type = token.Asterisk
	case '/':
		tok.Type = token.Slash
		if l.peek() == '/' {
			l.next()
			l.next()
			l.skipLineComment()
			return l.Next()
		}
		if l.peek() == '*' {
			l.next()
			l.next()
			l.skipBlockComment(tok.Position)
			return l.Next()
		}
	case '<':
		tok.Type = token.Less
		if l.peek() == '=' {
			l.next()
			tok.Type = token.LessEqual
		}
	case '>':
		tok.Type = token.Greater
		if l.peek() == '=' {
			l.next()
			tok.Type = token.GreaterEqual
		}
	case '!':
		tok.Type = token.Bang
		if l.peek() == '=' {
			l.next()
			tok.Type = token.NotEqual
		}
	case '?':
		tok.Type = token.Question
	case ':':
		tok.Type = token.Colon
	case '(':
		tok.Type = token.OpenParen
	case ')':
		tok.Type = token.CloseParen
	case '{':
		tok.Type = token.OpenBrace
	case '}':
		tok.Type = token.CloseBrace
	case '"':
		tok.Type = token.String
		tok.Literal = l.consumeString(tok.Position)
	default:
		l.errHandler(tok.Position, fmt.Sprintf("illegal character %#U", l.ch))
		tok.Type = token.Illegal
		tok.Literal = string(l.ch)
	}
	l.next()

	return tok
}

func (l *Lexer) skipWhitespace() {
	for isWhitespace(l.ch) {
		l.next()
	}
}

func (l *Lexer) skipLineComment() {
	for l.ch != '\n' && l.ch != eof {
		l.next()
	}
}

func (l *Lexer) skipBlockComment(startPos token.Position) {
	// Block comments can span multiple lines and they can also be nested
	openBlocks := 1 // There's already a block open when this method is called
	for openBlocks > 0 && l.ch != eof {
		if l.ch == '/' && l.peek() == '*' {
			l.next()
			openBlocks++
		} else if l.ch == '*' && l.peek() == '/' {
			l.next()
			openBlocks--
		}
		l.next()
	}
	if openBlocks > 0 {
		l.errHandler(startPos, "unterminated block comment")
	}
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

func (l *Lexer) consumeString(startPos token.Position) string {
	var b strings.Builder
	b.WriteRune('"')
	l.next()
loop:
	for {
		switch l.ch {
		case eof, '\n', '\r':
			l.errHandler(startPos, "unterminated string literal")
			break loop
		case '"':
			b.WriteRune(l.ch)
			break loop
		}
		b.WriteRune(l.ch)
		l.next()
	}
	return b.String()
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
	if l.readOffset == len(l.src) {
		l.ch = eof
		return
	}

	if l.ch == '\n' {
		l.pos.Line++
		l.pos.Column = 1
	} else {
		l.pos.Column++
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
