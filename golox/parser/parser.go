// Package parser implements a parser for Lox source code.
package parser

import (
	"fmt"
	"io"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/lox"
	"github.com/marcuscaisey/lox/golox/token"
)

const (
	maxParams = 255
	maxArgs   = maxParams
)

// Parse parses the source code read from r.
// If an error is returned then an incomplete AST will still be returned along with it.
func Parse(r io.Reader) (ast.Program, error) {
	l, err := newLexer(r)
	if err != nil {
		return ast.Program{}, fmt.Errorf("constructing parser: %s", err)
	}

	p := &parser{l: l}
	errHandler := func(tok token.Token, msg string) {
		p.lastErrPos = tok.Start
		p.errs.AddFromToken(tok, msg)
	}
	l.SetErrorHandler(errHandler)

	return p.Parse()
}

type parser struct {
	l          *lexer
	tok        token.Token // token currently being considered
	nextTok    token.Token
	loopDepth  int
	curFunType funType

	errs       lox.Errors
	lastErrPos token.Position
}

// Parse parses the source code and returns the root node of the abstract syntax tree.
// If an error is returned then an incomplete AST will still be returned along with it.
func (p *parser) Parse() (ast.Program, error) {
	// Populate tok and nextTok
	p.next()
	p.next()
	program := ast.Program{}
	for p.tok.Type != token.EOF {
		program.Stmts = append(program.Stmts, p.safelyParseDecl())
	}
	return program, p.errs.Err()
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
	switch tok := p.tok; {
	case p.match(token.Var):
		return p.parseVarDecl(tok)
	case p.tok.Type == token.Fun && p.nextTok.Type == token.Ident:
		p.match(token.Fun)
		return p.parseFunDecl(tok)
	case p.match(token.Class):
		return p.parseClassDecl(tok)
	default:
		return p.parseStmt()
	}
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
	params, body := p.parseFunParamsAndBody(funTypeFunction)
	return ast.FunDecl{
		Fun:        funTok,
		Name:       name,
		Params:     params,
		Body:       body.Stmts,
		RightBrace: body.RightBrace,
	}
}

func (p *parser) parseClassDecl(classTok token.Token) ast.ClassDecl {
	name := p.expectf(token.Ident, "expected class name")
	p.expect(token.LeftBrace)
	var methods []ast.MethodDecl
	for {
		name, ok := p.match2(token.Ident)
		if !ok {
			break
		}
		funType := funTypeMethod
		if name.Lexeme == token.InitIdent {
			funType = funTypeInit
		}
		params, body := p.parseFunParamsAndBody(funType)
		methods = append(methods, ast.MethodDecl{
			Name:       name,
			Params:     params,
			Body:       body.Stmts,
			RightBrace: body.RightBrace,
		})
	}
	rightBrace := p.expect(token.RightBrace)
	return ast.ClassDecl{
		Class:      classTok,
		Name:       name,
		Body:       methods,
		RightBrace: rightBrace,
	}
}

type funType int

const (
	funTypeNone funType = iota
	funTypeFunction
	funTypeMethod
	funTypeInit
)

func (p *parser) parseFunParamsAndBody(funType funType) ([]token.Token, ast.BlockStmt) {
	// Break and continue are not allowed to jump out of a function so reset the loop depth to catch any invalid uses.
	prevLoopDepth := p.loopDepth
	p.loopDepth = 0
	defer func() { p.loopDepth = prevLoopDepth }()

	prevFunType := p.curFunType
	p.curFunType = funType
	defer func() { p.curFunType = prevFunType }()

	p.expect(token.LeftParen)
	var params []token.Token
	if !p.match(token.RightParen) {
		params = p.parseParams()
		p.expect(token.RightParen)
	}
	leftBrace := p.expect(token.LeftBrace)
	body := p.parseBlock(leftBrace)
	return params, body
}

func (p *parser) parseParams() []token.Token {
	var params []token.Token
	seen := map[string]bool{}
	params = append(params, p.expectf(token.Ident, "expected parameter name"))
	seen[params[0].Lexeme] = true
	for p.match(token.Comma) {
		param := p.expectf(token.Ident, "expected parameter name")
		if seen[param.Lexeme] {
			p.addTokenError(param, "duplicate parameter %s", param.Lexeme)
		}
		params = append(params, param)
		if param.Lexeme != token.BlankIdent {
			seen[param.Lexeme] = true
		}
	}
	if len(params) > maxParams {
		p.addTokenError(params[maxParams], "cannot define more than %d function parameters", maxParams)
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
	var stmts []ast.Stmt
	for p.tok.Type != token.RightBrace && p.tok.Type != token.EOF {
		stmts = append(stmts, p.safelyParseDecl())
	}
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
	p.loopDepth++
	defer func() { p.loopDepth-- }()
	p.expect(token.LeftParen)
	condition := p.parseExpr()
	p.expect(token.RightParen)
	body := p.parseStmt()
	return ast.WhileStmt{While: whileTok, Condition: condition, Body: body}
}

func (p *parser) parseForStmt(forTok token.Token) ast.ForStmt {
	p.loopDepth++
	defer func() { p.loopDepth-- }()
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
	stmt := ast.BreakStmt{Break: breakTok, Semicolon: semicolon}
	if p.loopDepth == 0 {
		p.addNodeError(stmt, "%m can only be used inside a loop", token.Break)
	}
	return stmt
}

func (p *parser) parseContinueStmt(continueTok token.Token) ast.ContinueStmt {
	semicolon := p.expect(token.Semicolon)
	stmt := ast.ContinueStmt{Continue: continueTok, Semicolon: semicolon}
	if p.loopDepth == 0 {
		p.addNodeError(stmt, "%m can only be used inside a loop", token.Continue)
	}
	return stmt
}

func (p *parser) parseReturnStmt(returnTok token.Token) ast.ReturnStmt {
	semicolon, ok := p.match2(token.Semicolon)
	var value ast.Expr
	if !ok {
		value = p.parseExpr()
		semicolon = p.expect(token.Semicolon)
	}
	stmt := ast.ReturnStmt{Return: returnTok, Value: value, Semicolon: semicolon}
	if p.curFunType == funTypeNone {
		p.addNodeError(stmt, "%m can only be used inside a function definition", token.Return)
	}
	if p.curFunType == funTypeInit && stmt.Value != nil {
		p.addNodeError(stmt, "%s() cannot return a value", token.InitIdent)
	}
	return stmt
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
			p.addNodeError(expr, "invalid assignment target")
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
	args = append(args, p.parseAssignmentExpr())
	for p.match(token.Comma) {
		args = append(args, p.parseAssignmentExpr())
	}
	if len(args) > maxArgs {
		p.addNodeError(args[maxArgs], "cannot pass more than %d arguments to function", maxArgs)
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
		if p.curFunType != funTypeMethod && p.curFunType != funTypeInit {
			p.addTokenError(tok, "%m can only be used inside a method definition", token.This)
		}
		return ast.ThisExpr{This: tok}
	case p.match(token.Fun):
		return p.parseFunExpr(tok)
	case p.match(token.LeftParen):
		expr := p.parseExpr()
		rightParen := p.expect(token.RightParen)
		return ast.GroupExpr{LeftParen: tok, Expr: expr, RightParen: rightParen}
	// Error productions
	case p.match(token.EqualEqual, token.BangEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual, token.Asterisk, token.Slash, token.Plus):
		p.addTokenError(tok, "binary operator %m must have left and right operands", tok.Type)
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
		p.addTokenError(tok, "expected expression")
		panic(unwind{})
	}
}

func (p *parser) parseFunExpr(funTok token.Token) ast.FunExpr {
	params, body := p.parseFunParamsAndBody(funTypeFunction)
	return ast.FunExpr{
		Fun:        funTok,
		Params:     params,
		Body:       body.Stmts,
		RightBrace: body.RightBrace,
	}
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
	p.addTokenError(p.tok, format, a...)
	panic(unwind{})
}

// next advances the parser to the next token.
func (p *parser) next() {
	p.tok = p.nextTok
	p.nextTok = p.l.Next()
}

func (p *parser) addError(start token.Position, end token.Position, format string, args ...any) {
	if len(p.errs) > 0 && start == p.lastErrPos {
		return
	}
	p.errs.Add(start, end, format, args...)
}

func (p *parser) addTokenError(tok token.Token, format string, a ...any) {
	p.addError(tok.Start, tok.End, format, a...)
}

func (p *parser) addNodeError(node ast.Node, format string, a ...any) {
	p.addError(node.Start(), node.End(), format, a...)
}

// unwind is used as a panic value so that we can unwind the stack and recover from a parsing error without having to
// check for errors after every call to each parsing method.
type unwind struct{}
