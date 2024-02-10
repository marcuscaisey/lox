// Package parser implements a parser for Lox source code.
package parser

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/template"

	"github.com/fatih/color"
	"github.com/lithammer/dedent"
	"github.com/mattn/go-runewidth"

	"github.com/marcuscaisey/golox/ast"
	"github.com/marcuscaisey/golox/lexer"
	"github.com/marcuscaisey/golox/token"
)

// Parse parses the source code read from r.
// If an error is returned then an incomplete AST will still be returned along with it.
func Parse(r io.Reader) (ast.Node, error) {
	l, err := lexer.New(r)
	if err != nil {
		return ast.Program{}, fmt.Errorf("constructing parser: %s", err)
	}

	p := &parser{l: l}
	errHandler := func(tok token.Token, msg string) {
		p.lastErrPos = tok.Position
		err := &syntaxError{
			tok: tok,
			msg: msg,
		}
		p.errs = append(p.errs, err)
	}
	l.SetErrorHandler(errHandler)

	return p.Parse()
}

// parser parses Lox source code into an abstract syntax tree.
type parser struct {
	l          *lexer.Lexer
	tok        token.Token // token currently being considered
	errs       []error
	lastErrPos token.Position
}

// Parse parses the source code and returns the root node of the abstract syntax tree.
// If an error is returned then an incomplete AST will still be returned along with it.
func (p *parser) Parse() (ast.Node, error) {
	p.next() // Advance to the first token
	program := ast.Program{}
	for p.tok.Type != token.EOF {
		program.Stmts = append(program.Stmts, p.safelyParseDecl())
	}
	if len(p.errs) > 0 {
		return program, errors.Join(p.errs...)
	}
	return program, nil
}

func (p *parser) safelyParseDecl() (stmt ast.Stmt) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(unwind); ok {
				p.sync()
				stmt = ast.IllegalStmt{}
			} else {
				panic(r)
			}
		}
	}()
	return p.parseDecl()
}

// sync synchronises the parser with the next statement. This is used to recover from a parsing error.
func (p *parser) sync() {
	for p.tok.Type != token.EOF {
		switch p.tok.Type {
		case token.Semicolon:
			p.next()
			return
		case token.Print, token.Var:
			return
		}
		p.next()
	}
}

func (p *parser) parseDecl() ast.Stmt {
	switch {
	case p.match(token.Var):
		return p.parseVarDecl()
	default:
		return p.parseStmt()
	}
}

func (p *parser) parseVarDecl() ast.Stmt {
	name := p.expect(token.Ident, "%h must be followed by a variable name, found %h", token.Var, p.tok.Type)
	var value ast.Expr
	if p.match(token.Assign) {
		value = p.parseExpr()
	}
	p.expectSemicolon("variable declaration")
	return ast.VarDecl{Name: name, Initialiser: value}
}

func (p *parser) parseStmt() ast.Stmt {
	switch {
	case p.match(token.Print):
		return p.parsePrintStmt()
	default:
		return p.parseExprStmt()
	}
}

func (p *parser) parsePrintStmt() ast.Stmt {
	expr := p.parseExpr()
	p.expectSemicolon("print statement")
	return ast.PrintStmt{Expr: expr}
}

func (p *parser) parseExprStmt() ast.Stmt {
	expr := p.parseExpr()
	p.expectSemicolon("expression statement")
	return ast.ExprStmt{Expr: expr}
}

func (p *parser) parseExpr() ast.Expr {
	return p.parseCommaExpr()
}

func (p *parser) parseCommaExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseAssignExpr, token.Comma)
}

func (p *parser) parseAssignExpr() ast.Expr {
	expr := p.parseTernaryExpr()
	if left, ok := expr.(ast.VariableExpr); ok && p.match(token.Assign) {
		right := p.parseAssignExpr()
		expr = ast.AssignmentExpr{
			Left:  left.Name,
			Right: right,
		}
	}
	return expr
}

func (p *parser) parseTernaryExpr() ast.Expr {
	expr := p.parseEqualityExpr()
	if p.match(token.Question) {
		then := p.parseExpr()
		p.expect(token.Colon, "next part of ternary expression should be %h, found %h", token.Colon, p.tok.Type)
		elseExpr := p.parseTernaryExpr()
		expr = ast.TernaryExpr{
			Condition: expr,
			Then:      then,
			Else:      elseExpr,
		}
	}
	return expr
}

func (p *parser) parseEqualityExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseRelationalExpr, token.Equal, token.NotEqual)
}

func (p *parser) parseRelationalExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseAdditiveExpr, token.Less, token.LessEqual, token.Greater, token.GreaterEqual)
}

func (p *parser) parseAdditiveExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseMultiplicativeExpr, token.Plus, token.Minus)
}

func (p *parser) parseMultiplicativeExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseUnaryExpr, token.Asterisk, token.Slash)
}

// parseBinaryExpr parses a binary expression which uses the given operators. next is a function which parses an
// expression of next highest precedence.
func (p *parser) parseBinaryExpr(next func() ast.Expr, operators ...token.Type) ast.Expr {
	expr := next()
	for {
		op, ok := p.match2(operators...)
		if !ok {
			break
		}
		right := next()
		expr = ast.BinaryExpr{
			Left:  expr,
			Op:    op,
			Right: right,
		}
	}
	return expr
}

func (p *parser) parseUnaryExpr() ast.Expr {
	if op, ok := p.match2(token.Bang, token.Minus); ok {
		right := p.parseUnaryExpr()
		return ast.UnaryExpr{
			Op:    op,
			Right: right,
		}
	}
	return p.parsePrimaryExpr()
}

func (p *parser) parsePrimaryExpr() ast.Expr {
	var expr ast.Expr
	switch tok := p.tok; tok.Type {
	case token.Number:
		value, err := strconv.ParseFloat(p.tok.Literal, 64)
		if err != nil {
			panic(fmt.Sprintf("unexpected error parsing number literal: %s", err))
		}
		expr = ast.LiteralExpr{Value: value}
	case token.String:
		value := p.tok.Literal[1 : len(p.tok.Literal)-1] // Remove surrounding quotes
		expr = ast.LiteralExpr{Value: value}
	case token.True:
		expr = ast.LiteralExpr{Value: true}
	case token.False:
		expr = ast.LiteralExpr{Value: false}
	case token.Nil:
		expr = ast.LiteralExpr{Value: nil}
	case token.LeftParen:
		p.next()
		innerExpr := p.parseExpr()
		p.expect(token.RightParen, "expected closing %h after expression, found %h", token.RightParen, p.tok.Type)
		expr = ast.GroupExpr{Expr: innerExpr}
		return expr
	case token.Ident:
		expr = ast.VariableExpr{Name: p.tok}
	// Error productions
	case token.Equal, token.NotEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual, token.Asterisk, token.Slash, token.Plus:
		p.addSyntaxErrorf("binary operator %h must have left and right operands", p.tok.Type)
		p.next()
		var right ast.Expr
		switch tok.Type {
		case token.Equal, token.NotEqual:
			right = p.parseEqualityExpr()
		case token.Less, token.LessEqual, token.Greater, token.GreaterEqual:
			right = p.parseRelationalExpr()
		case token.Plus:
			right = p.parseMultiplicativeExpr()
		case token.Asterisk, token.Slash:
			right = p.parseUnaryExpr()
		}
		return ast.BinaryExpr{
			Left:  ast.IllegalExpr{},
			Op:    tok,
			Right: right,
		}
	default:
		p.addSyntaxErrorf("expected expression, found %h", p.tok.Type)
		panic(unwind{})
	}
	p.next()
	return expr
}

// match returns whether the current token is one of the given types and advances the parser if so.
func (p *parser) match(types ...token.Type) bool {
	for _, t := range types {
		if p.tok.Type == t {
			p.next()
			return true
		}
	}
	return false
}

// match2 is like match but also returns the matched token.
func (p *parser) match2(types ...token.Type) (token.Token, bool) {
	tok := p.tok
	return tok, p.match(types...)
}

// expect returns the current token and advances the parser if it has the given type. Otherwise, a syntax error is
// raised with the given format and arguments.
func (p *parser) expect(t token.Type, format string, a ...any) token.Token {
	if p.tok.Type == t {
		tok := p.tok
		p.next()
		return tok
	}
	p.addSyntaxErrorf(format, a...)
	panic(unwind{})
}

func (p *parser) expectSemicolon(context string) {
	p.expect(token.Semicolon, "expected %h after %s, found %h", token.Semicolon, context, p.tok.Type)
}

// next advances the parser to the next token.
func (p *parser) next() {
	p.tok = p.l.Next()
}

func (p *parser) addSyntaxErrorf(format string, a ...any) {
	if len(p.errs) > 0 && p.tok.Position == p.lastErrPos {
		return
	}
	p.errs = append(p.errs, &syntaxError{
		tok: p.tok,
		msg: fmt.Sprintf(format, a...),
	})
}

// unwind is used as a panic value so that we can unwind the stack and recover from a parsing error without having to
// check for errors after every call to each parsing method.
type unwind struct{}

type syntaxError struct {
	tok token.Token
	msg string
}

func (e *syntaxError) Error() string {
	line := e.tok.Position.File.Line(e.tok.Position.Line)

	var tok string
	switch {
	case e.tok.IsKeyword(), e.tok.IsSymbol():
		tok = e.tok.Type.String()
	case e.tok.IsLiteral():
		tok = e.tok.Literal
	case e.tok.Type == token.Illegal:
		if e.tok.Literal == "" {
			// If the token has no literal, then we don't have anything to point to.
			return fmt.Sprintf("%s: syntax error: %s", e.tok.Position, e.msg)
		}
		tok = e.tok.Literal
	case e.tok.Type == token.EOF:
		// We pretend that the EOF token is a space so that we have something to point to.
		tok = " "
		line = append(line, ' ')
	}
	// If the token spans multiple lines, only show the first one. I'm not sure what the best way of pointing to a
	// multi-line token is.
	tok, _, _ = strings.Cut(tok, "\n")

	data := map[string]any{
		"pos":    e.tok.Position,
		"msg":    e.msg,
		"before": string(line[:e.tok.Position.Column]),
		"tok":    tok,
		"after":  string(line[e.tok.Position.Column+len(tok):]),
	}
	funcs := template.FuncMap{
		"red":    color.New(color.FgRed).SprintFunc(),
		"bold":   color.New(color.Bold).SprintFunc(),
		"repeat": strings.Repeat,
		"stringWidth": func(s string) int {
			return runewidth.StringWidth(s)
		},
	}
	text := strings.TrimSpace(dedent.Dedent(`
		{{ .pos }}: syntax error: {{ .msg }}
		{{ .before }}{{ .tok | bold | red }}{{ .after }}
		{{ repeat " " (stringWidth .before) }}{{ repeat "^" (stringWidth .tok) | red | bold }}
	`))

	tmpl := template.Must(template.New("").Funcs(funcs).Parse(text))
	var b strings.Builder
	if err := tmpl.Execute(&b, data); err != nil {
		panic(err)
	}
	return b.String()
}
