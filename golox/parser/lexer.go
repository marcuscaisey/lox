package parser

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/marcuscaisey/lox/golox/token"
)

const eof = -1

// errorHandler is the function which handles syntax errors encountered during lexing.
// It's passed the offending token and a message describing the error.
type errorHandler func(tok token.Token, msg string)

// lexer converts Lox source code into lexical tokens.
// Tokens are read from the lexer using the Next method.
// Syntax errors are handled by calling the error handler function which can be set using SetErrorHandler. The default
// error handler is a no-op.
type lexer struct {
	// Immutable state
	src        []byte
	errHandler errorHandler

	// Mutable state
	ch           rune           // character currently being considered
	pos          token.Position // position of character currently being considered
	readOffset   int            // position of next character to be read
	lastReadSize int            // size of last rune read
}

// newLexer constructs a lexer which will lex the source code read from an io.Reader.
func newLexer(r io.Reader) (*lexer, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("constructing lexer: %s", err)
	}
	errHandler := func(token.Token, string) {}
	filename := name(r)

	l := &lexer{
		src:        src,
		errHandler: errHandler,
		pos: token.Position{
			File:   token.NewFile(filename, src),
			Line:   1,
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
func (l *lexer) SetErrorHandler(errHandler errorHandler) {
	l.errHandler = errHandler
}

// Next returns the next token. An EOF token is returned if the end of the source code has been reached.
func (l *lexer) Next() token.Token {
	l.skipWhitespace()

	tok := token.Token{Start: l.pos}

	switch {
	case l.ch == eof:
		tok.Type = token.EOF
	case l.ch == ';':
		tok.Type = token.Semicolon
	case l.ch == ',':
		tok.Type = token.Comma
	case l.ch == '.':
		tok.Type = token.Dot
	case l.ch == '=':
		tok.Type = token.Equal
		if l.peek() == '=' {
			l.next()
			tok.Type = token.EqualEqual
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
				tok.End = l.pos
				tok.Type = token.Illegal
				tok.Literal = comment
				l.errHandler(tok, "unterminated multi-line comment")
			}
			return l.Next()
		}
	case l.ch == '%':
		tok.Type = token.Percent
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
			tok.Type = token.BangEqual
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
		lit, terminated := l.consumeString()
		tok.End = l.pos
		tok.Literal = lit
		if terminated {
			tok.Type = token.String
		} else {
			tok.Type = token.Illegal
			l.errHandler(tok, "unterminated string literal")
		}
		return tok
	case isDigit(l.ch):
		tok.Type = token.Number
		tok.Literal = l.consumeNumber()
		tok.End = l.pos
		return tok
	case isAlpha(l.ch):
		ident := l.consumeIdent()
		tok.End = l.pos
		tok.Type = token.LookupIdent(ident)
		if tok.Type == token.Ident {
			tok.Literal = ident
		}
		return tok
	default:
		ch := l.ch
		l.next()
		tok.End = l.pos
		tok.Type = token.Illegal
		tok.Literal = string(ch)
		l.errHandler(tok, fmt.Sprintf("illegal character %#U", ch))
		return tok
	}

	l.next()
	tok.End = l.pos

	return tok
}

func (l *lexer) skipWhitespace() {
	for isWhitespace(l.ch) {
		l.next()
	}
}

func (l *lexer) skipSingleLineComment() {
	for l.ch != '\n' && l.ch != eof {
		l.next()
	}
}

func (l *lexer) consumeMultiLineComment() (comment string, terminated bool) {
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

func (l *lexer) consumeNumber() string {
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

func (l *lexer) consumeString() (s string, terminated bool) {
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

func (l *lexer) consumeIdent() string {
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
func (l *lexer) next() {
	if l.ch == eof {
		return
	}

	if l.ch == '\n' {
		l.pos.Line++
		l.pos.Column = 0
	} else {
		l.pos.Column += l.lastReadSize
	}

	if l.readOffset == len(l.src) {
		l.ch = eof
		return
	}

	r, size := utf8.DecodeRune(l.src[l.readOffset:])
	l.lastReadSize = size
	l.readOffset += size

	if r == utf8.RuneError {
		tok := token.Token{
			Start:   l.pos,
			End:     l.pos,
			Type:    token.Illegal,
			Literal: string(l.src[l.readOffset-1 : l.readOffset]),
		}
		tok.End.Column++
		l.errHandler(tok, fmt.Sprintf("invalid UTF-8 byte %#x", l.src[l.readOffset-1]))
		l.next()
		return
	}

	l.ch = r
}

// peek returns the next character without advancing the lexer.
// If the end of the source code has been reached, eof is returned.
func (l *lexer) peek() rune {
	if l.readOffset >= len(l.src) {
		return eof
	}
	return rune(l.src[l.readOffset])
}
