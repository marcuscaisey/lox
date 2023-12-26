// Package parser defines Parser which parses lexical tokens into an abstract syntax tree.
package parser

import (
	"errors"
	"fmt"

	"github.com/marcuscaisey/golox/ast"
	"github.com/marcuscaisey/golox/token"
)

// Parser parses lexical tokens into an abstract syntax tree.
type Parser struct {
	tokens    []token.Token
	pos       int
	curToken  token.Token
	nextToken token.Token
	errored   bool
	errors    []error
}

// New returns a new Parser.
func New(tokens []token.Token) *Parser {
	return &Parser{
		tokens:    tokens,
		nextToken: tokens[0],
	}
}

// Parse parses the lexical tokens into an abstract syntax tree.
func (p *Parser) Parse() (ast.Node, error) {
	expr := p.parseExpr()
	if len(p.errors) > 0 {
		return nil, errors.Join(p.errors...)
	}
	return expr, nil
}

func (p *Parser) parseExpr() ast.Expr {
	return p.parseEqualityExpr()
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
	if p.match(token.Number, token.String) {
		return ast.LiteralExpr{Value: p.curToken.Literal}
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
		if !p.expect(token.CloseParen) {
			return nil
		}
		return ast.GroupExpr{Expr: expr}
	}
	p.addSyntaxError("expected expression after %s, got %s", p.curToken, p.nextToken)
	return nil
}

// match returns whether the next token is any of the given types and advances the parser if it is.
// If the parser has errored then match will return false.
func (p *Parser) match(types ...token.Type) bool {
	if p.errored {
		return false
	}
	for _, t := range types {
		if p.nextToken.Type == t {
			p.advance()
			return true
		}
	}
	return false
}

// expect does the same as match but adds a syntax error if the next token is not the given type.
func (p *Parser) expect(t token.Type) bool {
	if p.errored {
		return false
	}
	if p.nextToken.Type == t {
		p.advance()
		return true
	}
	p.addSyntaxError("expected %s after %s, got %s", t, p.curToken, p.nextToken)
	return false
}

// advance increments the parser's position and updates the current and next tokens.
func (p *Parser) advance() {
	p.curToken = p.nextToken
	p.pos++
	p.nextToken = p.tokens[p.pos]
}

func (p *Parser) addSyntaxError(format string, a ...any) {
	msg := fmt.Sprintf(format, a...)
	p.errors = append(p.errors, fmt.Errorf("%s: syntax error: %s", p.nextToken.Pos, msg))
	p.errored = true
}
