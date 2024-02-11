// Package parser implements a parser for Lox source code.
package parser

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
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
		p.lastErrPos = tok.Start
		err := &syntaxError{
			start: tok.Start,
			end:   tok.End,
			msg:   msg,
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
	from := p.tok
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(unwind); ok {
				to := p.sync()
				stmt = ast.IllegalStmt{From: from, To: to}
			} else {
				panic(r)
			}
		}
	}()
	return p.parseDecl()
}

// sync synchronises the parser with the next statement. This is used to recover from a parsing error.
// The final token before the next statement is returned.
func (p *parser) sync() token.Token {
	finalTok := p.tok
	for {
		switch p.tok.Type {
		case token.Semicolon:
			finalTok := p.tok
			p.next()
			return finalTok
		case token.Print, token.Var, token.EOF:
			return finalTok
		}
		finalTok = p.tok
		p.next()
	}
}

func (p *parser) parseDecl() ast.Stmt {
	switch tok := p.tok; {
	case p.match(token.Var):
		return p.parseVarDecl(tok)
	default:
		return p.parseStmt()
	}
}

func (p *parser) parseVarDecl(varTok token.Token) ast.Stmt {
	name := p.expect(token.Ident, "%h must be followed by a variable name, found %h", token.Var, p.tok.Type)
	var value ast.Expr
	if p.match(token.Assign) {
		value = p.parseExpr()
	}
	semicolon := p.expectSemicolon("variable declaration")
	return ast.VarDecl{Var: varTok, Name: name, Initialiser: value, Semicolon: semicolon}
}

func (p *parser) parseStmt() ast.Stmt {
	switch tok := p.tok; {
	case p.match(token.Print):
		return p.parsePrintStmt(tok)
	default:
		return p.parseExprStmt()
	}
}

func (p *parser) parsePrintStmt(printTok token.Token) ast.Stmt {
	expr := p.parseExpr()
	semicolon := p.expectSemicolon("print statement")
	return ast.PrintStmt{Print: printTok, Expr: expr, Semicolon: semicolon}
}

func (p *parser) parseExprStmt() ast.Stmt {
	expr := p.parseExpr()
	semicolon := p.expectSemicolon("expression statement")
	return ast.ExprStmt{Expr: expr, Semicolon: semicolon}
}

func (p *parser) parseExpr() ast.Expr {
	return p.parseCommaExpr()
}

func (p *parser) parseCommaExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseAssignmentExpr, token.Comma)
}

func (p *parser) parseAssignmentExpr() ast.Expr {
	expr := p.parseTernaryExpr()
	if p.match(token.Assign) {
		left, ok := expr.(ast.VariableExpr)
		if !ok {
			p.addNodeErrorf(expr, "left-hand side of assignment must be a variable")
		}
		right := p.parseAssignmentExpr()
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
	case token.Number, token.String, token.True, token.False, token.Nil:
		expr = ast.LiteralExpr{Value: tok}
	case token.LeftParen:
		leftParen := tok
		p.next()
		innerExpr := p.parseExpr()
		rightParen := p.expect(token.RightParen, "expected closing %h after expression, found %h", token.RightParen, p.tok.Type)
		return ast.GroupExpr{LeftParen: leftParen, Expr: innerExpr, RightParen: rightParen}
	case token.Ident:
		expr = ast.VariableExpr{Name: tok}
	// Error productions
	case token.Equal, token.NotEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual, token.Asterisk, token.Slash, token.Plus:
		p.addTokenErrorf(p.tok, "binary operator %h must have left and right operands", tok.Type)
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
		p.addTokenErrorf(tok, "expected expression, found %h", tok.Type)
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
	p.addTokenErrorf(p.tok, format, a...)
	panic(unwind{})
}

func (p *parser) expectSemicolon(context string) token.Token {
	return p.expect(token.Semicolon, "expected %h after %s, found %h", token.Semicolon, context, p.tok.Type)
}

// next advances the parser to the next token.
func (p *parser) next() {
	p.tok = p.l.Next()
}

func (p *parser) addSyntaxErrorf(start, end token.Position, format string, a ...any) {
	if len(p.errs) > 0 && start == p.lastErrPos {
		return
	}
	p.errs = append(p.errs, &syntaxError{
		start: start,
		end:   end,
		msg:   fmt.Sprintf(format, a...),
	})
}

func (p *parser) addTokenErrorf(tok token.Token, format string, a ...any) {
	p.addSyntaxErrorf(tok.Start, tok.End, format, a...)
}

func (p *parser) addNodeErrorf(node ast.Node, format string, a ...any) {
	p.addSyntaxErrorf(node.Start(), node.End(), format, a...)
}

// unwind is used as a panic value so that we can unwind the stack and recover from a parsing error without having to
// check for errors after every call to each parsing method.
type unwind struct{}

type syntaxError struct {
	start token.Position
	end   token.Position
	msg   string
}

func (e *syntaxError) Error() string {
	bold := color.New(color.Bold)
	red := color.New(color.FgRed)

	var b strings.Builder
	bold.Fprint(&b, e.start, ": ", red.Sprint("syntax error: "), e.msg)

	firstLine := e.start.File.Line(e.start.Line)
	if !utf8.Valid(firstLine) {
		return b.String()
	}

	fmt.Fprintln(&b)

	if e.start.Line == e.end.Line {
		fmt.Fprintln(&b, string(firstLine))
		fmt.Fprint(&b, strings.Repeat(" ", runewidth.StringWidth(string(firstLine[:e.start.Column]))))
		red.Fprint(&b, strings.Repeat("~", runewidth.StringWidth(string(firstLine[e.start.Column:e.end.Column]))))
	} else {
		fmt.Fprintln(&b, string(firstLine))
		fmt.Fprint(&b, strings.Repeat(" ", runewidth.StringWidth(string(firstLine[:e.start.Column]))))
		red.Fprintln(&b, strings.Repeat("~", runewidth.StringWidth(string(firstLine[e.start.Column:]))))
		for i := e.start.Line + 1; i < e.end.Line; i++ {
			line := e.start.File.Line(i)
			fmt.Fprintln(&b, string(line))
			red.Fprintln(&b, strings.Repeat("~", runewidth.StringWidth(string(line))))
		}
		lastLine := e.start.File.Line(e.end.Line)
		fmt.Fprintln(&b, string(lastLine))
		red.Fprint(&b, strings.Repeat("~", runewidth.StringWidth(string(lastLine[:e.end.Column]))))
	}

	return b.String()
}
