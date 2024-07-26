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
	l            *lexer
	tok          token.Token // token currently being considered
	nextTok      token.Token // next token
	loopDepth    int
	funDeclDepth int

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
	default:
		return p.parseStmt()
	}
}

func (p *parser) parseVarDecl(varTok token.Token) ast.VarDecl {
	name := p.expectf(token.Ident, "%h must be followed by a variable name", token.Var)
	var value ast.Expr
	if p.match(token.Equal) {
		value = p.parseExpr("on right-hand side of variable declaration")
	}
	semicolon := p.expectSemicolon("after variable declaration")
	return ast.VarDecl{Var: varTok, Name: name, Initialiser: value, Semicolon: semicolon}
}

func (p *parser) parseFunDecl(funTok token.Token) ast.FunDecl {
	p.funDeclDepth++
	defer func() { p.funDeclDepth-- }()
	name := p.expectf(token.Ident, "expected function name")
	params, body := p.parseFunParamsAndBody()
	return ast.FunDecl{
		Fun:    funTok,
		Name:   name,
		Params: params,
		Body:   body,
	}
}

func (p *parser) parseFunParamsAndBody() ([]token.Token, []ast.Stmt) {
	leftParen := p.expectf(token.LeftParen, "expected parameter list inside %h%h", token.LeftParen, token.RightParen)
	var params []token.Token
	if !p.match(token.RightParen) {
		params = p.parseParams("for function parameter")
		// TODO: extract this out
		p.expectf(token.RightParen, "expected closing %h after opening %h at %s", token.RightParen, token.LeftParen, leftParen.Start)
	}
	leftBrace := p.expectf(token.LeftBrace, "expected block for function body")
	body := p.parseBlock(leftBrace)
	return params, body.Stmts
}

func (p *parser) parseParams(context string) []token.Token {
	var params []token.Token
	seen := map[string]bool{}
	params = append(params, p.expectf(token.Ident, "expected identifier %s", context))
	seen[params[0].Literal] = true
	for p.match(token.Comma) {
		param := p.expectf(token.Ident, "expected identifier %s", context)
		if seen[param.Literal] {
			p.addTokenError(param, "duplicate parameter %s", param.Literal)
		}
		params = append(params, param)
		seen[param.Literal] = true
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
	expr := p.parseExpr("in expression statement")
	semicolon := p.expectSemicolon("after expression statement")
	return ast.ExprStmt{Expr: expr, Semicolon: semicolon}
}

func (p *parser) parsePrintStmt(printTok token.Token) ast.PrintStmt {
	expr := p.parseExpr("in print statement")
	semicolon := p.expectSemicolon("after print statement")
	return ast.PrintStmt{Print: printTok, Expr: expr, Semicolon: semicolon}
}

func (p *parser) parseBlock(leftBrace token.Token) ast.BlockStmt {
	var stmts []ast.Stmt
	for p.tok.Type != token.RightBrace && p.tok.Type != token.EOF {
		stmts = append(stmts, p.safelyParseDecl())
	}
	rightBrace := p.expectf(token.RightBrace, "expected closing %h after block", token.RightBrace)
	return ast.BlockStmt{LeftBrace: leftBrace, Stmts: stmts, RightBrace: rightBrace}
}

func (p *parser) parseIfStmt(ifTok token.Token) ast.IfStmt {
	p.expectf(token.LeftParen, "%h should be followed by condition inside %h%h", token.If, token.LeftParen, token.RightParen)
	condition := p.parseExpr("as condition in if statement")
	p.expectf(token.RightParen, "%h should be followed by condition inside %h%h", token.If, token.LeftParen, token.RightParen)
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
	p.expectf(token.LeftParen, "%h should be followed by condition inside %h%h", token.While, token.LeftParen, token.RightParen)
	condition := p.parseExpr("as condition in while statement")
	p.expectf(token.RightParen, "%h should be followed by condition inside %h%h", token.While, token.LeftParen, token.RightParen)
	body := p.parseStmt()
	return ast.WhileStmt{While: whileTok, Condition: condition, Body: body}
}

func (p *parser) parseForStmt(forTok token.Token) ast.ForStmt {
	p.loopDepth++
	defer func() { p.loopDepth-- }()
	p.expectf(token.LeftParen, "%h should be followed by initialise statement, condition expression, and update expression inside %h%h", token.For, token.LeftParen, token.RightParen)
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
		condition = p.parseExpr("as condition in for loop")
		p.expectSemicolon("after for loop condition")
	}
	var update ast.Expr
	if !p.match(token.RightParen) {
		update = p.parseExpr("as update expression in for loop")
		p.expectf(token.RightParen, "%h should be followed by initialise statement, condition expression, and update expression inside %h%h", token.For, token.LeftParen, token.RightParen)
	}
	body := p.parseStmt()
	return ast.ForStmt{For: forTok, Initialise: initialise, Condition: condition, Update: update, Body: body}
}

func (p *parser) parseBreakStmt(breakTok token.Token) ast.BreakStmt {
	semicolon := p.expectSemicolon("after break statement")
	stmt := ast.BreakStmt{Break: breakTok, Semicolon: semicolon}
	if p.loopDepth == 0 {
		p.addNodeError(stmt, "break statement must be inside a loop")
	}
	return stmt
}

func (p *parser) parseContinueStmt(continueTok token.Token) ast.ContinueStmt {
	semicolon := p.expectSemicolon("after continue statement")
	stmt := ast.ContinueStmt{Continue: continueTok, Semicolon: semicolon}
	if p.loopDepth == 0 {
		p.addNodeError(stmt, "continue statement must be inside a loop")
	}
	return stmt
}

func (p *parser) parseReturnStmt(returnTok token.Token) ast.ReturnStmt {
	semicolon, ok := p.match2(token.Semicolon)
	var value ast.Expr
	if !ok {
		value = p.parseExpr("after return")
		semicolon = p.expectSemicolon("after return statement")
	}
	stmt := ast.ReturnStmt{Return: returnTok, Value: value, Semicolon: semicolon}
	if p.funDeclDepth == 0 {
		p.addNodeError(stmt, "return statement must be inside a function definition")
	}
	return stmt
}

func (p *parser) parseExpr(context string) ast.Expr {
	return p.parseCommaExpr(context)
}

func (p *parser) parseCommaExpr(context string) ast.Expr {
	return p.parseBinaryExpr(context, p.parseAssignmentExpr, token.Comma)
}

func (p *parser) parseAssignmentExpr(context string) ast.Expr {
	expr := p.parseTernaryExpr(context)
	if p.match(token.Equal) {
		left, ok := expr.(ast.VariableExpr)
		if !ok {
			p.addNodeError(expr, "left-hand side of assignment must be a variable")
		}
		right := p.parseAssignmentExpr("on right-hand side of assignment")
		expr = ast.AssignmentExpr{
			Left:  left.Name,
			Right: right,
		}
	}
	return expr
}

func (p *parser) parseTernaryExpr(context string) ast.Expr {
	expr := p.parseLogicalOrExpr(context)
	if p.match(token.Question) {
		then := p.parseExpr("for then part of ternary expression")
		p.expectf(token.Colon, "next part of ternary expression should be %h", token.Colon)
		elseExpr := p.parseTernaryExpr("for else part of ternary expression")
		expr = ast.TernaryExpr{
			Condition: expr,
			Then:      then,
			Else:      elseExpr,
		}
	}
	return expr
}

func (p *parser) parseLogicalOrExpr(context string) ast.Expr {
	return p.parseBinaryExpr(context, p.parseLogicalAndExpr, token.Or)
}

func (p *parser) parseLogicalAndExpr(context string) ast.Expr {
	return p.parseBinaryExpr(context, p.parseEqualityExpr, token.And)
}

func (p *parser) parseEqualityExpr(context string) ast.Expr {
	return p.parseBinaryExpr(context, p.parseRelationalExpr, token.EqualEqual, token.BangEqual)
}

func (p *parser) parseRelationalExpr(context string) ast.Expr {
	return p.parseBinaryExpr(context, p.parseAdditiveExpr, token.Less, token.LessEqual, token.Greater, token.GreaterEqual)
}

func (p *parser) parseAdditiveExpr(context string) ast.Expr {
	return p.parseBinaryExpr(context, p.parseMultiplicativeExpr, token.Plus, token.Minus)
}

func (p *parser) parseMultiplicativeExpr(context string) ast.Expr {
	return p.parseBinaryExpr(context, p.parseUnaryExpr, token.Asterisk, token.Slash, token.Percent)
}

// parseBinaryExpr parses a binary expression which uses the given operators. next is a function which parses an
// expression of next highest precedence.
func (p *parser) parseBinaryExpr(context string, next func(context string) ast.Expr, operators ...token.Type) ast.Expr {
	expr := next(context)
	for {
		op, ok := p.match2(operators...)
		if !ok {
			break
		}
		right := next("on right-hand side of binary expression")
		expr = ast.BinaryExpr{
			Left:  expr,
			Op:    op,
			Right: right,
		}
	}
	return expr
}

func (p *parser) parseUnaryExpr(context string) ast.Expr {
	if op, ok := p.match2(token.Bang, token.Minus); ok {
		right := p.parseUnaryExpr("after unary operator")
		return ast.UnaryExpr{
			Op:    op,
			Right: right,
		}
	}
	return p.parseCallExpr(context)
}

func (p *parser) parseCallExpr(context string) ast.Expr {
	expr := p.parsePrimaryExpr(context)
	for {
		leftParen, ok := p.match2(token.LeftParen)
		if !ok {
			return expr
		}
		var args []ast.Expr
		rightParen, ok := p.match2(token.RightParen)
		if !ok {
			args = p.parseArgs("for function argument")
			rightParen = p.expectf(token.RightParen, "expected closing %h after opening %h at %s", token.RightParen, token.LeftParen, leftParen.Start)
		}
		expr = ast.CallExpr{
			Callee:     expr,
			Args:       args,
			RightParen: rightParen,
		}
	}
}

func (p *parser) parseArgs(context string) []ast.Expr {
	var args []ast.Expr
	args = append(args, p.parseAssignmentExpr(context))
	for p.match(token.Comma) {
		args = append(args, p.parseAssignmentExpr(context))
	}
	if len(args) > maxArgs {
		p.addNodeError(args[maxArgs], "cannot pass more than %d arguments to function", maxArgs)
	}
	return args
}

func (p *parser) parsePrimaryExpr(context string) ast.Expr {
	switch tok := p.tok; {
	case p.match(token.Number, token.String, token.True, token.False, token.Nil):
		return ast.LiteralExpr{Value: tok}
	case p.match(token.Ident):
		return ast.VariableExpr{Name: tok}
	case p.match(token.Fun):
		return p.parseFunExpr(tok)
	case p.match(token.LeftParen):
		innerExpr := p.parseExpr("inside parentheses")
		rightParen := p.expectf(token.RightParen, "expected closing %h after opening %h at %s", token.RightParen, token.LeftParen, tok.Start)
		return ast.GroupExpr{LeftParen: tok, Expr: innerExpr, RightParen: rightParen}
	// Error productions
	case p.match(token.EqualEqual, token.BangEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual, token.Asterisk, token.Slash, token.Plus):
		p.addTokenError(tok, "binary operator %h must have left and right operands", tok.Type)
		var right ast.Expr
		switch tok.Type {
		case token.EqualEqual, token.BangEqual:
			right = p.parseEqualityExpr("on right-hand side of binary expression")
		case token.Less, token.LessEqual, token.Greater, token.GreaterEqual:
			right = p.parseRelationalExpr("on right-hand side of binary expression")
		case token.Plus:
			right = p.parseMultiplicativeExpr("on right-hand side of binary expression")
		case token.Asterisk, token.Slash:
			right = p.parseUnaryExpr("on right-hand side of binary expression")
		}
		return ast.BinaryExpr{
			Op:    tok,
			Right: right,
		}
	default:
		p.addTokenError(tok, "expected expression "+context)
		panic(unwind{})
	}
}

func (p *parser) parseFunExpr(funTok token.Token) ast.FunExpr {
	p.funDeclDepth++
	defer func() { p.funDeclDepth-- }()
	params, body := p.parseFunParamsAndBody()
	return ast.FunExpr{
		Fun:    funTok,
		Params: params,
		Body:   body,
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

// expectf returns the current token and advances the parser if it has the given type. Otherwise, a syntax error is
// raised with the given format and arguments.
func (p *parser) expectf(t token.Type, format string, a ...any) token.Token {
	if p.tok.Type == t {
		tok := p.tok
		p.next()
		return tok
	}
	p.addTokenError(p.tok, format, a...)
	panic(unwind{})
}

func (p *parser) expectSemicolon(context string) token.Token {
	return p.expectf(token.Semicolon, "expected %h %s", token.Semicolon, context)
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
