// Package lexer implements a lexer for Lox source code.
package lexer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/marcuscaisey/golox/token"
)

const nullChar = 0

// Lexer converts Lox source code into lexical tokens.
type Lexer struct {
	file      *token.File
	src       string
	startPos  int // position of the first character of the lexeme being scanned
	pos       int // position of the character currently being considered
	startLine int // line of the first character of the lexeme being scanned
	line      int // line of the character currently being considered
	startCol  int // column of the first character of the lexeme being scanned
	// BUG: This is not correct for multi-byte (e.g. UTF-8) characters.
	col int // column of the character currently being considered
}

// New constructs a Lexer which will lex the provided source code.
func New(src string) *Lexer {
	return &Lexer{
		file: token.NewFile(src),
		src:  src,
		line: 1,
		col:  1,
	}
}

// Lex converts the source code into a sequences of tokens.
func (s *Lexer) Lex() ([]token.Token, error) {
	var tokens []token.Token
	var errs []error
	for {
		nextToken, err := s.consumeToken()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		tokens = append(tokens, nextToken)
		if nextToken.Type == token.EOF {
			break
		}
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return tokens, nil
}

func (s *Lexer) consumeToken() (token.Token, error) {
	s.consumeWhitespace()
	s.startPos = s.pos
	s.startLine = s.line
	s.startCol = s.col
	switch char := s.consumeChar(); char {
	case nullChar:
		return s.newToken(token.EOF), nil
	case ';':
		return s.newToken(token.Semicolon), nil
	case ',':
		return s.newToken(token.Comma), nil
	case '.':
		return s.newToken(token.Dot), nil
	case '=':
		if s.peekChar() == '=' {
			s.consumeChar()
			return s.newToken(token.Equal), nil
		}
		return s.newToken(token.Assign), nil
	case '+':
		return s.newToken(token.Plus), nil
	case '-':
		return s.newToken(token.Minus), nil
	case '*':
		return s.newToken(token.Asterisk), nil
	case '/':
		if s.peekChar() == '/' {
			s.consumeChar()
			s.consumeLineComment()
			return s.consumeToken()
		}
		if s.peekChar() == '*' {
			s.consumeChar()
			if err := s.consumeBlockComment(); err != nil {
				return token.Token{}, err
			}
			return s.consumeToken()
		}
		return s.newToken(token.Slash), nil
	case '<':
		if s.peekChar() == '=' {
			s.consumeChar()
			return s.newToken(token.LessEqual), nil
		}
		return s.newToken(token.Less), nil
	case '>':
		if s.peekChar() == '=' {
			s.consumeChar()
			return s.newToken(token.GreaterEqual), nil
		}
		return s.newToken(token.Greater), nil
	case '!':
		if s.peekChar() == '=' {
			s.consumeChar()
			return s.newToken(token.NotEqual), nil
		}
		return s.newToken(token.Bang), nil
	case '?':
		return s.newToken(token.Question), nil
	case ':':
		return s.newToken(token.Colon), nil
	case '(':
		return s.newToken(token.OpenParen), nil
	case ')':
		return s.newToken(token.CloseParen), nil
	case '{':
		return s.newToken(token.OpenBrace), nil
	case '}':
		return s.newToken(token.CloseBrace), nil
	case '"':
		return s.consumeStringToken()
	default:
		if isDigit(char) {
			return s.consumeNumberToken(), nil
		}
		if isAlpha(char) {
			ident := s.consumeIdent()
			tokenType := token.LookupIdent(ident)
			return s.newToken(tokenType), nil
		}
		return token.Token{}, s.errorf("unexpected character %q", char)
	}
}

// consumeChar returns the character at the current position and advances it if EOF has not been reached. Otherwise,
// nullChar is returned.
func (s *Lexer) consumeChar() byte {
	if s.eofReached() {
		return nullChar
	}
	char := s.src[s.pos]
	s.pos++
	s.col++
	return char
}

// peekChar returns the character at the current position without advancing it if EOF has not been reached. Otherwise,
// nullChar is returned.
func (s *Lexer) peekChar() byte {
	if s.eofReached() {
		return nullChar
	}
	return s.src[s.pos]
}

// peekNextChar returns the character after the current position without consuming it.
// If EOF has been reached, nullChar is returned.
func (s *Lexer) peekNextChar() byte {
	if s.pos >= len(s.src)-1 {
		return nullChar
	}
	return s.src[s.pos+1]
}

func (s *Lexer) eofReached() bool {
	return s.pos >= len(s.src)
}

func (s *Lexer) consumeWhitespace() {
	for isWhitespace(s.peekChar()) {
		if s.consumeChar() == '\n' {
			s.line++
			s.col = 1
		}
	}
}

func (s *Lexer) consumeLineComment() {
	for !s.eofReached() && s.peekChar() != '\n' {
		s.consumeChar()
	}
}

func (s *Lexer) consumeBlockComment() error {
	// Block comments can span multiple lines and they can also be nested
	openBlocks := 1 // There's already a block open when this method is called
	for openBlocks > 0 && !s.eofReached() {
		s.consumeWhitespace()
		if s.peekChar() == '/' && s.peekNextChar() == '*' {
			s.consumeChar()
			s.consumeChar()
			openBlocks++
		} else if s.peekChar() == '*' && s.peekNextChar() == '/' {
			s.consumeChar()
			s.consumeChar()
			openBlocks--
		} else {
			s.consumeChar()
		}
	}
	if openBlocks > 0 {
		return s.errorf("unterminated block comment: %s", s.scannedLexeme())
	}
	return nil
}

func (s *Lexer) consumeStringToken() (token.Token, error) {
	for {
		switch s.consumeChar() {
		case nullChar, '\n', '\r':
			replacer := strings.NewReplacer(
				"\n", ``,
				"\r", ``,
			)
			return token.Token{}, s.errorf("unterminated string literal: %s", replacer.Replace(s.scannedLexeme()))
		case '"':
			return s.newToken(token.String), nil
		}
	}
}

func (s *Lexer) consumeNumberToken() token.Token {
	for isDigit(s.peekChar()) {
		s.consumeChar()
	}
	if s.peekChar() == '.' && isDigit(s.peekNextChar()) {
		s.consumeChar()
		s.consumeChar()
		for isDigit(s.peekChar()) {
			s.consumeChar()
		}
	}
	return s.newToken(token.Number)
}

func (s *Lexer) consumeIdent() string {
	for isAlphaNumeric(s.peekChar()) {
		s.consumeChar()
	}
	return s.scannedLexeme()
}

func isWhitespace(char byte) bool {
	switch char {
	case ' ', '\r', '\t', '\n':
		return true
	default:
		return false
	}
}

func isDigit(char byte) bool {
	return '0' <= char && char <= '9'
}

func isAlpha(char byte) bool {
	return ('a' <= char && char <= 'z') || ('A' <= char && char <= 'Z') || char == '_'
}

func isAlphaNumeric(char byte) bool {
	return isAlpha(char) || isDigit(char)
}

func (s *Lexer) scannedLexeme() string {
	return s.src[s.startPos:s.pos]
}

// newToken returns a Token with its Literal, Line, and Column set based on the current state of the Lexer.
func (s *Lexer) newToken(tokenType token.Type) token.Token {
	return token.Token{
		Type:    tokenType,
		Literal: s.scannedLexeme(),
		Pos: token.Position{
			File:   s.file,
			Line:   s.line,
			Column: s.startCol,
		},
	}
}

func (s *Lexer) errorf(format string, a ...any) error {
	msg := fmt.Sprintf(format, a...)
	return fmt.Errorf("%d:%d: syntax error: %s", s.startLine, s.startCol, msg)
}