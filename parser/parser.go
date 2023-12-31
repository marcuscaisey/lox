// Package parser implements a parser for Lox source code.
package parser

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/marcuscaisey/golox/ast"
	"github.com/marcuscaisey/golox/lexer"
	"github.com/marcuscaisey/golox/token"
)

var ansiCodes = map[string]string{
	"RESET":   "\x1b[0m",
	"BOLD":    "\x1b[1m",
	"RED":     "\x1b[31m",
	"DEFAULT": "\x1b[39m",
}

var isTerminal = term.IsTerminal(int(os.Stderr.Fd()))

type syntaxError struct {
	pos token.Position
	msg string
}

func (e *syntaxError) Error() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("${BOLD}%s: ${RED}syntax error: ${DEFAULT}%s${RESET}\n", e.pos, e.msg))
	line := e.pos.File.Line(e.pos.Line)
	before := line[:e.pos.Column-1]
	after := line[e.pos.Column:]
	// TODO: highlight the entire token
	b.WriteString(fmt.Sprintf("%s${RED}%s${RESET}%s\n", before, string(line[e.pos.Column-1]), after))
	b.WriteString(fmt.Sprintf("${BOLD}${RED}%*s${RESET}", e.pos.Column, "^"))
	msg := b.String()
	for k, v := range ansiCodes {
		if !isTerminal {
			v = ""
		}
		msg = strings.ReplaceAll(msg, fmt.Sprintf("${%s}", k), v)
	}
	return msg
}

// Parser parses Lox source code into an abstract syntax tree.
type Parser struct {
	l           *lexer.Lexer
	tok         token.Token // token currently being considered
	errs        []error
	lastErrLine int // line number of last error
}

// New constructs a new Parser which parses the source code read from r.
func New(r io.Reader) (*Parser, error) {
	l, err := lexer.New(r)
	if err != nil {
		return nil, fmt.Errorf("constructing parser: %s", err)
	}

	p := &Parser{l: l}

	errHandler := func(pos token.Position, msg string) {
		p.lastErrLine = pos.Line
		err := &syntaxError{
			pos: pos,
			msg: msg,
		}
		p.errs = append(p.errs, err)
	}
	l.SetErrorHandler(errHandler)

	p.next()

	return p, nil
}

// Parse parses the source code and returns the root node of the abstract syntax tree.
func (p *Parser) Parse() (ast.Node, error) {
	root := p.safelyParseExpr()
	if len(p.errs) > 0 {
		return nil, errors.Join(p.errs...)
	}
	return root, nil
}

func (p *Parser) safelyParseExpr() ast.Expr {
	defer func() {
		if r := recover(); r != nil {
			if syntaxErr, ok := r.(*syntaxError); ok {
				if len(p.errs) > 0 && syntaxErr.pos.Line == p.lastErrLine {
					return
				}
				p.errs = append(p.errs, syntaxErr)
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
	if p.tok.Type == token.Question {
		p.next()
		then := p.parseExpr()
		p.expect(token.Colon)
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
	for slices.Contains(operators, p.tok.Type) {
		op := p.tok.Type
		p.next()
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
	if p.tok.Type == token.Bang || p.tok.Type == token.Minus {
		op := p.tok.Type
		p.next()
		right := p.parseUnaryExpr()
		return ast.UnaryExpr{
			Op:    op,
			Right: right,
		}
	}
	return p.parsePrimaryExpr()
}

func (p *Parser) parsePrimaryExpr() ast.Expr {
	var primaryExpr ast.Expr
	switch p.tok.Type {
	case token.Number:
		value, err := strconv.ParseFloat(p.tok.Literal, 64)
		if err != nil {
			panic(fmt.Sprintf("unexpected error parsing number literal: %s", err))
		}
		primaryExpr = ast.LiteralExpr{Value: value}
	case token.String:
		value := p.tok.Literal[1 : len(p.tok.Literal)-1] // Remove surrounding quotes
		primaryExpr = ast.LiteralExpr{Value: value}
	case token.True:
		primaryExpr = ast.LiteralExpr{Value: true}
	case token.False:
		primaryExpr = ast.LiteralExpr{Value: false}
	case token.Nil:
		primaryExpr = ast.LiteralExpr{Value: nil}
	case token.LeftParen:
		p.next()
		expr := p.parseExpr()
		p.expect(token.RightParen)
		primaryExpr = ast.GroupExpr{Expr: expr}
	default:
		p.errorf("expected expression, got %s", p.tok)
	}
	p.next()
	return primaryExpr
}

// expect checks that the current token is of the given type and calls p.next() if so. Otherwise, an error is raised.
func (p *Parser) expect(t token.Type) {
	if p.tok.Type == t {
		p.next()
		return
	}
	p.errorf("expected %s, got %s", t, p.tok)
}

// next reads the next token from the lexer into p.tok.
func (p *Parser) next() {
	p.tok = p.l.Next()
}

func (p *Parser) errorf(format string, a ...any) {
	err := &syntaxError{
		msg: fmt.Sprintf(format, a...),
		pos: p.tok.Position,
	}
	panic(err)
}
