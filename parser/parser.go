// Package parser defines Parser which parses lexical tokens into an abstract syntax tree.
package parser

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/marcuscaisey/golox/ast"
	"github.com/marcuscaisey/golox/token"
)

var ansiCodes = map[string]string{
	"RESET":   "\x1b[0m",
	"BOLD":    "\x1b[1m",
	"RED":     "\x1b[31m",
	"DEFAULT": "\x1b[39m",
}

var isTerminal = term.IsTerminal(int(os.Stderr.Fd()))

type parserError struct {
	error
}

type syntaxError struct {
	msg string
	pos token.Position
}

func (e *syntaxError) Error() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("${BOLD}%s: ${RED}syntax error: ${DEFAULT}%s${RESET}\n", e.pos, e.msg))
	b.WriteString(fmt.Sprintf("%s\n", e.pos.File.Line(e.pos.Line)))
	b.WriteString(fmt.Sprintf("${RED}${BOLD}%*s${RESET}", e.pos.Column, "^"))
	msg := b.String()
	for k, v := range ansiCodes {
		if !isTerminal {
			v = ""
		}
		msg = strings.ReplaceAll(msg, fmt.Sprintf("${%s}", k), v)
	}
	return msg
}

// Parser parses lexical tokens into an abstract syntax tree.
type Parser struct {
	tokens    []token.Token
	pos       int
	curToken  token.Token
	nextToken token.Token
	errors    []error
}

// New returns a new Parser.
func New(tokens []token.Token) *Parser {
	return &Parser{
		tokens:    tokens,
		nextToken: tokens[0],
	}
}

// Parse parses its tokens and returns the root node of the abstract syntax tree.
func (p *Parser) Parse() (ast.Node, error) {
	root := p.safelyParseExpr()
	if len(p.errors) > 0 {
		return nil, errors.Join(p.errors...)
	}
	return root, nil
}

func (p *Parser) safelyParseExpr() ast.Expr {
	defer func() {
		if r := recover(); r != nil {
			if parserErr, ok := r.(parserError); ok {
				p.errors = append(p.errors, parserErr)
			} else {
				panic(r)
			}
		}
	}()
	return p.parseExpr()
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseCommaExpr()
}

func (p *Parser) parseCommaExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseTernaryExpr, token.Comma)
}

func (p *Parser) parseTernaryExpr() ast.Expr {
	expr := p.parseEqualityExpr()
	if p.match(token.Question) {
		then := p.parseExpr()
		p.mustMatch(token.Colon)
		elseExpr := p.parseTernaryExpr()
		expr = ast.TernaryExpr{
			Condition: expr,
			Then:      then,
			Else:      elseExpr,
		}
	}
	return expr
}

func (p *Parser) parseEqualityExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseRelationalExpr, token.Equal, token.NotEqual)
}

func (p *Parser) parseRelationalExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseAdditiveExpr, token.Less, token.LessEqual, token.Greater, token.GreaterEqual)
}

func (p *Parser) parseAdditiveExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseMultiplicativeExpr, token.Plus, token.Minus)
}

func (p *Parser) parseMultiplicativeExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseUnaryExpr, token.Asterisk, token.Slash)
}

// parseBinaryExpr parses a binary expression which uses the given operators. next is a function which parses an
// expression of next highest precedence.
func (p *Parser) parseBinaryExpr(next func() ast.Expr, operators ...token.Type) ast.Expr {
	expr := next()
	for p.match(operators...) {
		op := p.curToken.Type
		right := next()
		expr = ast.BinaryExpr{
			Left:  expr,
			Op:    op,
			Right: right,
		}
	}
	return expr
}

func (p *Parser) parseUnaryExpr() ast.Expr {
	if p.match(token.Bang, token.Minus) {
		op := p.curToken.Type
		right := p.parseUnaryExpr()
		return ast.UnaryExpr{
			Op:    op,
			Right: right,
		}
	}
	return p.parsePrimaryExpr()
}

func (p *Parser) parsePrimaryExpr() ast.Expr {
	if p.match(token.Number) {
		value, err := strconv.ParseFloat(p.curToken.Literal, 64)
		if err != nil {
			panic(fmt.Sprintf("unexpected error parsing number literal: %s", err))
		}
		return ast.LiteralExpr{Value: value}
	}
	if p.match(token.String) {
		value := p.curToken.Literal[1 : len(p.curToken.Literal)-1] // Remove surrounding quotes
		return ast.LiteralExpr{Value: value}
	}
	if p.match(token.True) {
		return ast.LiteralExpr{Value: true}
	}
	if p.match(token.False) {
		return ast.LiteralExpr{Value: false}
	}
	if p.match(token.Nil) {
		return ast.LiteralExpr{Value: nil}
	}
	if p.match(token.OpenParen) {
		expr := p.parseExpr()
		p.mustMatch(token.CloseParen)
		return ast.GroupExpr{Expr: expr}
	}
	p.error("expected expression after %s, got %s", p.curToken, p.nextToken)
	return nil
}

// match returns whether the next token is any of the given types and advances the parser if it is.
func (p *Parser) match(types ...token.Type) bool {
	for _, t := range types {
		if p.nextToken.Type == t {
			p.advance()
			return true
		}
	}
	return false
}

// mustMatch does the same as match but raises an error if the next token is not the given type.
func (p *Parser) mustMatch(t token.Type) {
	if p.match(t) {
		return
	}
	p.error("expected %s after %s, got %s", t, p.curToken, p.nextToken)
}

// advance increments the parser's position and updates the current and next tokens.
func (p *Parser) advance() {
	p.curToken = p.nextToken
	p.pos++
	p.nextToken = p.tokens[p.pos]
}

func (p *Parser) error(format string, a ...any) {
	err := &syntaxError{
		msg: fmt.Sprintf(format, a...),
		pos: p.nextToken.Position,
	}
	panic(parserError{err})
}
