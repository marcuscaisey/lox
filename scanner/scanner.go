// Package scanner defines Scanner which scans Lox source code into a sequence of lexical tokens.
package scanner

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/marcuscaisey/golox/token"
)

const nullChar = 0

type errorSlice []error

func (e errorSlice) Error() string {
	errStrings := make([]string, len(e))
	for i, err := range e {
		errStrings[i] = err.Error()
	}
	return strings.Join(errStrings, "\n")
}

// syntaxError is the error generated by the Scanner when it encounters invalid syntax.
type syntaxError struct {
	Line int
	Col  int
	Msg  string
}

func (e *syntaxError) Error() string {
	return fmt.Sprintf("%d:%d: syntax error: %s", e.Line, e.Col, e.Msg)
}

// Scanner scans Lox source code into lexical tokens.
type Scanner struct {
	src      string
	startPos int // position of the first character of the lexeme being scanned
	pos      int // position of the character currently being considered
	line     int // current line in the source file
	startCol int // column of the first character of the lexeme being scanned
	col      int // current column in the source file
}

// New constructs a Scanner which will scan the provided source code.
func New(src string) *Scanner {
	return &Scanner{
		src:  src,
		line: 1,
		col:  1,
	}
}

// Scan scans the source code into a sequences of tokens.
// If a syntax error is detected, an error is returned as well as any tokens which were successfully scanned.
func (s *Scanner) Scan() ([]token.Token, error) {
	var tokens []token.Token
	var errs errorSlice
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
		return tokens, errs
	}
	return tokens, nil
}

func (s *Scanner) consumeToken() (token.Token, error) {
	s.consumeWhitespace()
	s.startPos = s.pos
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
		return s.newToken(token.Astrisk), nil
	case '/':
		// // indicates a comment, so we should consume the rest of the line but ignore its contents
		if s.peekChar() == '/' {
			for s.peekChar() != '\n' {
				s.consumeChar()
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
		return s.newToken(token.Greater), nil
	case '!':
		if s.peekChar() == '=' {
			s.consumeChar()
			return s.newToken(token.NotEqual), nil
		}
		return s.newToken(token.Not), nil
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
		if isAlpha(char) || char == '_' {
			ident := s.consumeIdent()
			tokenType := token.LookupIdent(ident)
			return s.newToken(tokenType), nil
		}
		return token.Token{}, s.newSyntaxErrorf("unexpected character %s", string(char))
	}
}

// consumeChar consumes the character at the current position and returns it.
// If EOF has been reached, nullChar is returned.
// The current position is advanced if a character is returned.
func (s *Scanner) consumeChar() byte {
	if s.eofReached() {
		return nullChar
	}
	char := s.src[s.pos]
	s.pos++
	s.col++
	return char
}

// peekChar returns the character at the current position without consuming it.
// If EOF has been reached, nullChar is returned.
func (s *Scanner) peekChar() byte {
	if s.eofReached() {
		return nullChar
	}
	return s.src[s.pos]
}

// peekNextChar returns the character after the current position without consuming it.
// If EOF has been reached, nullChar is returned.
func (s *Scanner) peekNextChar() byte {
	if s.pos >= len(s.src)-1 {
		return nullChar
	}
	return s.src[s.pos+1]
}

func (s *Scanner) eofReached() bool {
	return s.pos >= len(s.src)
}

// consumeWhitespace consumes characters until either EOF or a non-whitespace character has been reached.
func (s *Scanner) consumeWhitespace() {
	for !s.eofReached() {
		if isWhitespace(s.peekChar()) {
			if s.consumeChar() == '\n' {
				s.line++
				s.col = 1
			}
		} else {
			return
		}
	}
}

func isWhitespace(char byte) bool {
	switch char {
	case ' ', '\r', '\t', '\n':
		return true
	default:
		return false
	}
}

func (s *Scanner) consumeStringToken() (token.Token, error) {
	for {
		switch s.consumeChar() {
		case nullChar, '\n', '\r':
			replacer := strings.NewReplacer(
				"\n", ``,
				"\r", ``,
			)
			return token.Token{}, s.newSyntaxErrorf("unterminated string literal: %s", replacer.Replace(s.scannedLexeme()))
		case '"':
			lexeme := s.scannedLexeme()
			literal := lexeme[1 : len(lexeme)-2] // trim off leading and trailing "
			return s.newTokenWithLiteral(token.String, literal), nil
		}
	}
}

func (s *Scanner) consumeNumberToken() token.Token {
	for isDigit(s.peekChar()) {
		s.consumeChar()
	}
	if s.peekChar() == '.' && isDigit(s.peekNextChar()) {
		s.consumeChar()
		for isDigit(s.peekChar()) {
			s.consumeChar()
		}
	}
	literal, err := strconv.ParseFloat(s.scannedLexeme(), 64)
	if err != nil {
		panic(fmt.Sprintf("Parsing of number literal should never fail. Error: %s", err))
	}
	return s.newTokenWithLiteral(token.Number, literal)
}

func isDigit(char byte) bool {
	return '0' <= char && char <= '9'
}

func (s *Scanner) consumeIdent() string {
	for isAlphaNumeric(s.peekChar()) {
		s.consumeChar()
	}
	return s.scannedLexeme()
}

func isAlpha(c byte) bool {
	return ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') || c == '_'
}

func isAlphaNumeric(c byte) bool {
	return isAlpha(c) || isDigit(c)
}

// scannedLexeme returns the portion of the current lexeme that has been scanned.
func (s *Scanner) scannedLexeme() string {
	return s.src[s.startPos:s.pos]
}

// newToken returns a Token with its Lexeme, Line, and Col set based on the current state of the Scanner.
func (s *Scanner) newToken(tokenType token.Type) token.Token {
	return token.Token{
		Type:   tokenType,
		Lexeme: s.scannedLexeme(),
		Line:   s.line,
		Col:    s.startCol,
	}
}

// newTokenWithLiteral returns a Token with its Lexeme, Line, and Col set based on the current state of the Scanner.
func (s *Scanner) newTokenWithLiteral(tokenType token.Type, literal any) token.Token {
	return token.Token{
		Type:    tokenType,
		Lexeme:  s.scannedLexeme(),
		Literal: literal,
		Line:    s.line,
		Col:     s.startCol,
	}
}

// newSyntaxErrorf returns a syntaxError with it Line and Col set based on the current state of the Scanner.
func (s *Scanner) newSyntaxErrorf(format string, a ...any) *syntaxError {
	return &syntaxError{
		Line: s.line,
		Col:  s.startCol,
		Msg:  fmt.Sprintf(format, a...),
	}
}
