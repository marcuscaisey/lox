package parser

import (
	"io"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/marcuscaisey/lox/golox/token"
)

const eof = -1

// errorHandler is the function which handles syntax errors encountered during lexing.
// It's passed the offending token and a format string and arguments to construct an error message from.
type errorHandler func(tok token.Token, format string, args ...any)

// lexer converts Lox source code into lexical tokens.
// Tokens are read from the lexer using the Next method.
// Syntax errors are handled by calling the error handler function which can be set using SetErrorHandler. The default
// error handler is a no-op.
type lexer struct {
	extraFeatures bool

	src        []byte
	errHandler errorHandler

	ch           rune           // character currently being considered
	pos          token.Position // position of character currently being considered
	offset       int            // offset of character currently being considered
	readOffset   int            // offset of next character to be read
	lastReadSize int            // size of last rune read
}

// newLexer constructs a lexer which will lex the source code read from an io.Reader.
// filename is the name of the file being lexed.
func newLexer(r io.Reader, filename string) (*lexer, error) {
	src, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	l := &lexer{
		extraFeatures: true,
		src:           src,
		errHandler:    func(token.Token, string, ...any) {},
		pos: token.Position{
			File:   token.NewFile(filename, src),
			Line:   1,
			Column: 0,
		},
	}

	l.next()

	return l, nil
}

// SetErrorHandler sets the error handler function which will be called when a syntax error is encountered.
func (l *lexer) SetErrorHandler(errHandler errorHandler) {
	l.errHandler = errHandler
}

// Next returns the next token. An EOF token is returned if the end of the source code has been reached.
func (l *lexer) Next() token.Token {
	l.skipWhitespace()

	startOffset := l.offset
	tok := token.Token{StartPos: l.pos}

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
		if l.peek() == '/' {
			tok.Type = token.Comment
			tok.Lexeme = l.consumeSingleLineComment()
			tok.EndPos = l.pos
			return tok
		} else {
			tok.Type = token.Slash
			break
		}
	case l.extraFeatures && l.ch == '%':
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
	case l.ch == '?' && l.extraFeatures:
		tok.Type = token.Question
	case l.ch == ':' && l.extraFeatures:
		tok.Type = token.Colon
	case l.ch == '(':
		tok.Type = token.LeftParen
	case l.ch == ')':
		tok.Type = token.RightParen
	case l.ch == '[' && l.extraFeatures:
		tok.Type = token.LeftBrack
	case l.ch == ']' && l.extraFeatures:
		tok.Type = token.RightBrack
	case l.ch == '{':
		tok.Type = token.LeftBrace
	case l.ch == '}':
		tok.Type = token.RightBrace
	case l.ch == '"':
		lit, terminated := l.consumeString()
		tok.EndPos = l.pos
		tok.Lexeme = lit
		if terminated {
			tok.Type = token.String
		} else {
			tok.Type = token.Illegal
			l.errHandler(tok, "unterminated string literal")
		}
		return tok
	case isDigit(l.ch):
		tok.Type = token.Number
		tok.Lexeme = l.consumeNumber()
		tok.EndPos = l.pos
		return tok
	case isAlpha(l.ch):
		ident := l.consumeIdent()
		tok.EndPos = l.pos
		tok.Type = token.IdentType(ident)
		if !l.extraFeatures && slices.Contains([]token.Type{token.Break, token.Continue, token.Static, token.Get, token.Set}, tok.Type) {
			tok.Type = token.Ident
		}
		tok.Lexeme = ident
		return tok
	default:
		ch := l.ch
		l.next()
		tok.EndPos = l.pos
		tok.Type = token.Illegal
		tok.Lexeme = string(ch)
		if unicode.IsPrint(ch) {
			l.errHandler(tok, "illegal character %c", ch)
		} else {
			l.errHandler(tok, "illegal character %#U", ch)
		}
		return tok
	}

	l.next()
	tok.EndPos = l.pos
	tok.Lexeme = string(l.src[startOffset:l.offset])

	return tok
}

func (l *lexer) skipWhitespace() {
	for isWhitespace(l.ch) {
		l.next()
	}
}

func (l *lexer) consumeSingleLineComment() string {
	l.next() // /
	l.next() // /
	var b strings.Builder
	b.WriteString("//")
	for l.ch != '\n' && l.ch != eof {
		b.WriteRune(l.ch)
		l.next()
	}
	return b.String()
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
	var b strings.Builder
	b.WriteRune('"')
	l.next()
	for {
		switch {
		case l.ch == eof:
			return b.String(), false
		case l.ch == '\\' && l.extraFeatures:
			s := l.consumeEscapeSequence()
			b.WriteString(s)
			continue
		}
		b.WriteRune(l.ch)
		ch := l.ch
		l.next()
		if ch == '"' {
			return b.String(), true
		}
	}
}

func (l *lexer) consumeEscapeSequence() string {
	var b strings.Builder
	b.WriteRune('\\')
	l.next()
	switch l.ch {
	case 'n', 't', '\\':
		b.WriteRune(l.ch)
		l.next()
		return b.String()
	case 'x':
		b.WriteRune(l.ch)
		l.next()
		if isHexDigit(l.ch) {
			b.WriteRune(l.ch)
			l.next()
		} else {
			tok := l.consumeIllegalToken()
			l.errHandler(tok, "invalid hex digit in escape sequence")
			b.WriteString(tok.Lexeme)
			return b.String()
		}
		if isHexDigit(l.ch) {
			b.WriteRune(l.ch)
			l.next()
		} else {
			tok := l.consumeIllegalToken()
			l.errHandler(tok, "invalid hex digit in escape sequence")
			b.WriteString(tok.Lexeme)
			return b.String()
		}
		return b.String()
	default:
		tok := l.consumeIllegalToken()
		l.errHandler(tok, "invalid escape sequence")
		b.WriteString(tok.Lexeme)
		return b.String()
	}
}

func isHexDigit(digit rune) bool {
	switch {
	case '0' <= digit && digit <= '9':
		return true
	case 'a' <= digit && digit <= 'f':
		return true
	case 'A' <= digit && digit <= 'F':
		return true
	default:
		return false
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

func (l *lexer) consumeIllegalToken() token.Token {
	startPos := l.pos
	startOffset := l.offset
	l.next()
	return token.Token{
		StartPos: startPos,
		EndPos:   l.pos,
		Type:     token.Illegal,
		Lexeme:   string(l.src[startOffset:l.offset]),
	}
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

	l.offset = l.readOffset

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

	if l.extraFeatures {
		r, size := utf8.DecodeRune(l.src[l.readOffset:])
		l.lastReadSize = size
		l.readOffset += size
		if r == utf8.RuneError {
			// If we get here then we've read exactly one invalid UTF-8 byte
			endPos := l.pos
			endPos.Column++
			tok := token.Token{
				StartPos: l.pos,
				EndPos:   l.pos,
				Type:     token.Illegal,
				Lexeme:   string(l.src[l.offset : l.offset+1]),
			}
			tok.EndPos.Column++
			l.errHandler(tok, "invalid UTF-8 byte %#x", tok.Lexeme)
			l.next()
			return
		}
		l.ch = r

	} else {
		l.ch = rune(l.src[l.readOffset])
		l.lastReadSize = 1
		l.readOffset++
	}
}

// peek returns the next character without advancing the lexer.
// If the end of the source code has been reached, eof is returned.
func (l *lexer) peek() rune {
	if l.readOffset >= len(l.src) {
		return eof
	}
	return rune(l.src[l.readOffset])
}
