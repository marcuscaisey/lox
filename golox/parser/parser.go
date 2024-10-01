// Package parser implements a parser for Lox source code.
package parser

import (
	"fmt"
	"io"
	"slices"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/lox"
	"github.com/marcuscaisey/lox/golox/token"
)

// Option can be passed to [Parse] to configure its behaviour.
type Option func(*parser)

// WithComments enables the parsing of comments.
func WithComments() Option {
	return func(p *parser) {
		p.parseComments = true
	}
}

// Parse parses the source code read from r.
// If an error is returned then an incomplete AST will still be returned along with it.
func Parse(r io.Reader, opts ...Option) (ast.Program, error) {
	lexer, err := newLexer(r)
	if err != nil {
		return ast.Program{}, fmt.Errorf("constructing parser: %s", err)
	}

	p := &parser{lexer: lexer}
	lexer.SetErrorHandler(func(tok token.Token, format string, args ...any) {
		p.addErrorf(lox.FromToken(tok), format, args...)
	})
	for _, opt := range opts {
		opt(p)
	}

	return p.Parse()
}

type parser struct {
	lexer   *lexer
	tok     token.Token // token currently being considered
	nextTok token.Token

	errs       lox.Errors
	lastErrPos token.Position

	parseComments bool
}

// Parse parses the source code and returns the root node of the abstract syntax tree.
// If an error is returned then an incomplete AST will still be returned along with it.
func (p *parser) Parse() (ast.Program, error) {
	// Populate tok and nextTok
	p.next()
	p.next()
	return p.parseProgram(), p.errs.Err()
}

func (p *parser) parseProgram() ast.Program {
	return ast.Program{
		Stmts: p.parseDeclsUntil(token.EOF),
	}
}

func (p *parser) parseDeclsUntil(types ...token.Type) []ast.Stmt {
	var stmts []ast.Stmt
	for !slices.Contains(types, p.tok.Type) {
		stmt := p.safelyParseDecl()
		if _, ok := stmt.(ast.CommentStmt); ok && !p.parseComments {
			continue
		}
		stmts = append(stmts, stmt)
	}
	return stmts
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
		case token.Print, token.Var, token.If, token.LeftBrace, token.While, token.For, token.Break, token.Continue, token.EOF:
			return finalTok
		}
		finalTok = p.tok
		p.next()
	}
}

func (p *parser) parseDecl() ast.Stmt {
	var stmt ast.Stmt
	switch tok := p.tok; {
	case p.match(token.Comment):
		stmt = p.parseCommentStmt(tok)
	case p.match(token.Var):
		stmt = p.parseVarDecl(tok)
	case p.tok.Type == token.Fun && p.nextTok.Type == token.Ident:
		p.match(token.Fun)
		stmt = p.parseFunDecl(tok)
	case p.match(token.Class):
		stmt = p.parseClassDecl(tok)
	default:
		stmt = p.parseStmt()
	}

	if comment, ok := p.matchFunc(func(tok token.Token) bool {
		return tok.Type == token.Comment && tok.Start.Line == stmt.End().Line
	}); ok && p.parseComments {
		return ast.InlineCommentStmt{Stmt: stmt, Comment: comment}
	}

	return stmt
}

func (p *parser) parseCommentStmt(commentTok token.Token) ast.CommentStmt {
	return ast.CommentStmt{Comment: commentTok}
}

func (p *parser) parseVarDecl(varTok token.Token) ast.VarDecl {
	name := p.expectf(token.Ident, "expected variable name")
	var value ast.Expr
	if p.match(token.Equal) {
		value = p.parseExpr()
	}
	semicolon := p.expect(token.Semicolon)
	return ast.VarDecl{Var: varTok, Name: name, Initialiser: value, Semicolon: semicolon}
}

func (p *parser) parseFunDecl(funTok token.Token) ast.FunDecl {
	name := p.expectf(token.Ident, "expected function name")
	return ast.FunDecl{
		Fun:      funTok,
		Name:     name,
		Function: p.parseFun(),
	}
}

func (p *parser) parseClassDecl(classTok token.Token) ast.ClassDecl {
	name := p.expectf(token.Ident, "expected class name")
	p.expect(token.LeftBrace)
	var body []ast.Stmt
	for {
		for {
			tok, ok := p.match2(token.Comment)
			if !ok {
				break
			}
			comment := p.parseCommentStmt(tok)
			if p.parseComments {
				body = append(body, comment)
			}
		}

		if decl, ok := p.parseMethodDecl(); ok {
			body = append(body, decl)
		} else {
			break
		}
	}
	rightBrace := p.expect(token.RightBrace)
	return ast.ClassDecl{
		Class:      classTok,
		Name:       name,
		Body:       body,
		RightBrace: rightBrace,
	}
}

func (p *parser) parseMethodDecl() (ast.MethodDecl, bool) {
	var modifiers []token.Token
	if tok, ok := p.match2(token.Static); ok {
		modifiers = append(modifiers, tok)
	}
	if tok, ok := p.match2(token.Get, token.Set); ok {
		modifiers = append(modifiers, tok)
	}

	var name token.Token
	if len(modifiers) > 0 {
		name = p.expectf(token.Ident, "expected method name")
	} else if tok, ok := p.match2(token.Ident); ok {
		name = tok
	} else {
		return ast.MethodDecl{}, false
	}

	return ast.MethodDecl{
		Modifiers: modifiers,
		Name:      name,
		Function:  p.parseFun(),
	}, true
}

func (p *parser) parseFun() ast.Function {
	leftParen := p.expect(token.LeftParen)
	var params []token.Token
	if !p.match(token.RightParen) {
		params = p.parseParams()
		p.expect(token.RightParen)
	}
	leftBrace := p.expect(token.LeftBrace)
	body := p.parseBlock(leftBrace)
	return ast.Function{
		LeftParen: leftParen,
		Params:    params,
		Body:      body,
	}
}

func (p *parser) parseParams() []token.Token {
	var params []token.Token
	for {
		params = append(params, p.expectf(token.Ident, "expected parameter name"))
		if !p.match(token.Comma) {
			break
		}
	}
	return params
}

func (p *parser) parseStmt() ast.Stmt {
	switch tok := p.tok; {
	case p.match(token.Print):
		return p.parsePrintStmt(tok)
	case p.match(token.LeftBrace):
		return p.parseBlock(tok)
	case p.match(token.If):
		return p.parseIfStmt(tok)
	case p.match(token.While):
		return p.parseWhileStmt(tok)
	case p.match(token.For):
		return p.parseForStmt(tok)
	case p.match(token.Break):
		return p.parseBreakStmt(tok)
	case p.match(token.Continue):
		return p.parseContinueStmt(tok)
	case p.match(token.Return):
		return p.parseReturnStmt(tok)
	default:
		return p.parseExprStmt()
	}
}

func (p *parser) parseExprStmt() ast.ExprStmt {
	expr := p.parseExpr()
	semicolon := p.expect(token.Semicolon)
	return ast.ExprStmt{Expr: expr, Semicolon: semicolon}
}

func (p *parser) parsePrintStmt(printTok token.Token) ast.PrintStmt {
	expr := p.parseExpr()
	semicolon := p.expect(token.Semicolon)
	return ast.PrintStmt{Print: printTok, Expr: expr, Semicolon: semicolon}
}

func (p *parser) parseBlock(leftBrace token.Token) ast.BlockStmt {
	stmts := p.parseDeclsUntil(token.RightBrace, token.EOF)
	rightBrace := p.expect(token.RightBrace)
	return ast.BlockStmt{LeftBrace: leftBrace, Stmts: stmts, RightBrace: rightBrace}
}

func (p *parser) parseIfStmt(ifTok token.Token) ast.IfStmt {
	p.expect(token.LeftParen)
	condition := p.parseExpr()
	p.expect(token.RightParen)
	thenBranch := p.parseStmt()
	var elseBranch ast.Stmt
	if p.match(token.Else) {
		elseBranch = p.parseStmt()
	}
	return ast.IfStmt{If: ifTok, Condition: condition, Then: thenBranch, Else: elseBranch}
}

func (p *parser) parseWhileStmt(whileTok token.Token) ast.WhileStmt {
	p.expect(token.LeftParen)
	condition := p.parseExpr()
	p.expect(token.RightParen)
	body := p.parseStmt()
	return ast.WhileStmt{While: whileTok, Condition: condition, Body: body}
}

func (p *parser) parseForStmt(forTok token.Token) ast.ForStmt {
	p.expect(token.LeftParen)
	var initialise ast.Stmt
	switch tok := p.tok; {
	case p.match(token.Var):
		initialise = p.parseVarDecl(tok)
	case p.match(token.Semicolon):
	default:
		initialise = p.parseExprStmt()
	}
	var condition ast.Expr
	if !p.match(token.Semicolon) {
		condition = p.parseExpr()
		p.expect(token.Semicolon)
	}
	var update ast.Expr
	if !p.match(token.RightParen) {
		update = p.parseExpr()
		p.expect(token.RightParen)
	}
	body := p.parseStmt()
	return ast.ForStmt{For: forTok, Initialise: initialise, Condition: condition, Update: update, Body: body}
}

func (p *parser) parseBreakStmt(breakTok token.Token) ast.BreakStmt {
	semicolon := p.expect(token.Semicolon)
	return ast.BreakStmt{Break: breakTok, Semicolon: semicolon}
}

func (p *parser) parseContinueStmt(continueTok token.Token) ast.ContinueStmt {
	semicolon := p.expect(token.Semicolon)
	return ast.ContinueStmt{Continue: continueTok, Semicolon: semicolon}
}

func (p *parser) parseReturnStmt(returnTok token.Token) ast.ReturnStmt {
	semicolon, ok := p.match2(token.Semicolon)
	var value ast.Expr
	if !ok {
		value = p.parseExpr()
		semicolon = p.expect(token.Semicolon)
	}
	return ast.ReturnStmt{Return: returnTok, Value: value, Semicolon: semicolon}
}

func (p *parser) parseExpr() ast.Expr {
	return p.parseCommaExpr()
}

func (p *parser) parseCommaExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseAssignmentExpr, token.Comma)
}

func (p *parser) parseAssignmentExpr() ast.Expr {
	expr := p.parseTernaryExpr()
	if p.match(token.Equal) {
		switch left := expr.(type) {
		case ast.VariableExpr:
			right := p.parseAssignmentExpr()
			expr = ast.AssignmentExpr{
				Left:  left.Name,
				Right: right,
			}
		case ast.GetExpr:
			right := p.parseAssignmentExpr()
			expr = ast.SetExpr{
				Object: left.Object,
				Name:   left.Name,
				Value:  right,
			}
		default:
			p.addError(lox.FromNode(expr), "invalid assignment target")
		}
	}
	return expr
}

func (p *parser) parseTernaryExpr() ast.Expr {
	expr := p.parseLogicalOrExpr()
	if p.match(token.Question) {
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

func (p *parser) parseLogicalOrExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseLogicalAndExpr, token.Or)
}

func (p *parser) parseLogicalAndExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseEqualityExpr, token.And)
}

func (p *parser) parseEqualityExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseRelationalExpr, token.EqualEqual, token.BangEqual)
}

func (p *parser) parseRelationalExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseAdditiveExpr, token.Less, token.LessEqual, token.Greater, token.GreaterEqual)
}

func (p *parser) parseAdditiveExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseMultiplicativeExpr, token.Plus, token.Minus)
}

func (p *parser) parseMultiplicativeExpr() ast.Expr {
	return p.parseBinaryExpr(p.parseUnaryExpr, token.Asterisk, token.Slash, token.Percent)
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
	return p.parseCallExpr()
}

func (p *parser) parseCallExpr() ast.Expr {
	expr := p.parsePrimaryExpr()
	for {
		switch {
		case p.match(token.LeftParen):
			var args []ast.Expr
			rightParen, ok := p.match2(token.RightParen)
			if !ok {
				args = p.parseArgs()
				rightParen = p.expect(token.RightParen)
			}
			expr = ast.CallExpr{
				Callee:     expr,
				Args:       args,
				RightParen: rightParen,
			}
		case p.match(token.Dot):
			name := p.expectf(token.Ident, "expected property name")
			expr = ast.GetExpr{
				Object: expr,
				Name:   name,
			}
		default:
			return expr
		}
	}
}

func (p *parser) parseArgs() []ast.Expr {
	var args []ast.Expr
	for {
		args = append(args, p.parseAssignmentExpr())
		if !p.match(token.Comma) {
			break
		}
	}
	return args
}

func (p *parser) parsePrimaryExpr() ast.Expr {
	switch tok := p.tok; {
	case p.match(token.Number, token.String, token.True, token.False, token.Nil):
		return ast.LiteralExpr{Value: tok}
	case p.match(token.Ident):
		return ast.VariableExpr{Name: tok}
	case p.match(token.This):
		return ast.ThisExpr{This: tok}
	case p.match(token.Fun):
		return p.parseFunExpr(tok)
	case p.match(token.LeftParen):
		expr := p.parseExpr()
		rightParen := p.expect(token.RightParen)
		return ast.GroupExpr{LeftParen: tok, Expr: expr, RightParen: rightParen}
	// Error productions
	case p.match(token.EqualEqual, token.BangEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual, token.Asterisk, token.Slash, token.Plus):
		p.addErrorf(lox.FromToken(tok), "binary operator %m must have left and right operands", tok.Type)
		var right ast.Expr
		switch tok.Type {
		case token.EqualEqual, token.BangEqual:
			right = p.parseEqualityExpr()
		case token.Less, token.LessEqual, token.Greater, token.GreaterEqual:
			right = p.parseRelationalExpr()
		case token.Plus:
			right = p.parseMultiplicativeExpr()
		case token.Asterisk, token.Slash:
			right = p.parseUnaryExpr()
		}
		return ast.BinaryExpr{
			Op:    tok,
			Right: right,
		}
	default:
		if tok.Type == token.Comment {
			p.addError(lox.FromToken(tok), "comments can only appear where declarations can appear or at the end of statements")
		} else {
			p.addError(lox.FromToken(tok), "expected expression")
		}
		panic(unwind{})
	}
}

func (p *parser) parseFunExpr(funTok token.Token) ast.FunExpr {
	return ast.FunExpr{
		Fun:      funTok,
		Function: p.parseFun(),
	}
}

// match reports whether the current token is one of the given types and advances the parser if so.
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

// matchFunc reports whether the current token satisfies the given predicate and advances the parser if so.
func (p *parser) matchFunc(f func(token.Token) bool) (token.Token, bool) {
	tok := p.tok
	if f(tok) {
		p.next()
		return tok, true
	}
	return tok, false
}

// expect returns the current token and advances the parser if it has the given type. Otherwise, an "expected %m" error
// is added and the method panics to unwind the stack.
func (p *parser) expect(t token.Type) token.Token {
	return p.expectf(t, "expected %m", t)
}

// expectf is like expect but accepts a format string for the error message.
func (p *parser) expectf(t token.Type, format string, a ...any) token.Token {
	if p.tok.Type == t {
		tok := p.tok
		p.next()
		return tok
	}
	p.addErrorf(lox.FromToken(p.tok), format, a...)
	panic(unwind{})
}

// next advances the parser to the next token.
func (p *parser) next() {
	p.tok = p.nextTok
	p.nextTok = p.lexer.Next()
}

func (p *parser) addError(rang lox.ErrorRange, message string) {
	p.addErrorf(rang, "%s", message)
}

func (p *parser) addErrorf(rang lox.ErrorRange, format string, args ...any) {
	start, _ := rang()
	if len(p.errs) > 0 && start == p.lastErrPos {
		return
	}
	p.lastErrPos = start
	p.errs.Addf(rang, format, args...)
}

// unwind is used as a panic value so that we can unwind the stack and recover from a parsing error without having to
// check for errors after every call to each parsing method.
type unwind struct{}
