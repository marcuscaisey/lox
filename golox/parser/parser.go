// Package parser implements a parser of Lox code.
package parser

import (
	"fmt"
	"io"
	"slices"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerr"
	"github.com/marcuscaisey/lox/golox/token"
)

// Option can be passed to [Parse] to configure its behaviour.
type Option func(*parser)

// WithComments enables the parsing of comments.
func WithComments(enabled bool) Option {
	return func(p *parser) {
		p.parseComments = enabled
	}
}

// WithPrintTokens enables the printing of tokens.
func WithPrintTokens(enabled bool) Option {
	return func(p *parser) {
		p.printTokens = enabled
	}
}

// WithExtraFeatures enables extra features that https://github.com/marcuscaisey/lox implements but the base Lox
// language does not.
// Extra features are enabled by default.
func WithExtraFeatures(enabled bool) Option {
	return func(p *parser) {
		p.extraFeatures = enabled
		p.lexer.extraFeatures = enabled
	}
}

// Parse parses the source code read from r.
// filename is the name of the file being parsed.
// If an error is returned then an incomplete program will still be returned along with it. If there are syntax errors
// then this error will be a [loxerr.Errors] containing all of the errors.
func Parse(r io.Reader, filename string, opts ...Option) (*ast.Program, error) {
	lexer, err := newLexer(r, filename)
	if err != nil {
		return nil, fmt.Errorf("parsing lox source: %w", err)
	}

	p := &parser{
		extraFeatures:       true,
		lexer:               lexer,
		classBodyScopeDepth: -1,
	}
	lexer.SetErrorHandler(func(tok token.Token, format string, args ...any) {
		p.addErrorf(tok, format, args...)
	})
	for _, opt := range opts {
		opt(p)
	}

	return p.Parse()
}

type parser struct {
	parseComments bool
	printTokens   bool
	extraFeatures bool

	lexer   *lexer
	prevTok token.Token
	tok     token.Token // token currently being considered
	nextTok token.Token

	scopeDepth          int
	classBodyScopeDepth int
	curClassDecl        *ast.ClassDecl
	midStmtComments     []*ast.Comment

	errs       loxerr.Errors
	lastErrPos token.Position
}

// Parse parses the source code and returns the root node of the abstract syntax tree.
// If an error is returned then an incomplete AST will still be returned along with it. If there are syntax errors then
// this error will be a [loxerr.Errors] containing all of the errors.
func (p *parser) Parse() (*ast.Program, error) {
	// Populate tok and nextTok
	p.next()
	p.next()
	return p.parseProgram(), p.errs.Err()
}

func (p *parser) parseProgram() *ast.Program {
	stmts := p.parseDeclsUntil(token.EOF)
	endPos := p.tok.End() // p.tok will be EOF.
	startPos := endPos
	startPos.Line = 1
	startPos.Column = 0
	return &ast.Program{StartPos: startPos, Stmts: stmts, EndPos: endPos}
}

func (p *parser) parseDeclsUntil(types ...token.Type) []ast.Stmt {
	var stmts []ast.Stmt
	var docComments []*ast.Comment
	for !slices.Contains(types, p.tok.Type) {
		from := p.tok
		stmt, ok := p.parseDecl()
		if !ok {
			to := p.sync()
			if stmt == nil {
				stmt = &ast.IllegalStmt{From: from, To: to}
			}
		}

		if len(docComments) > 0 && stmt.Start().Line != docComments[len(docComments)-1].Start().Line+1 {
			docComments = docComments[:0]
		}
		if comment, ok := stmt.(*ast.Comment); ok {
			if !p.parseComments {
				continue
			}
			docComments = append(docComments, comment)
		}

		resetDocComments := true
		switch decl := stmt.(type) {
		case *ast.FunDecl:
			decl.DocComments = docComments
		case *ast.ClassDecl:
			decl.DocComments = docComments
		case *ast.MethodDecl:
			decl.DocComments = docComments
		default:
			resetDocComments = false
		}
		if resetDocComments {
			stmts = stmts[:len(stmts)-len(docComments)]
			docComments = nil
		}
		stmts = append(stmts, stmt)
		if len(p.midStmtComments) > 0 {
			for _, comment := range p.midStmtComments {
				stmts = append(stmts, comment)
			}
			p.midStmtComments = p.midStmtComments[:0]
		}
	}
	return stmts
}

// sync synchronises the parser so that p.tok is positioned at the start of the next statement.
// The final token before the next statement is returned.
func (p *parser) sync() token.Token {
	finalTok := p.tok
	for {
		switch p.tok.Type {
		case token.Semicolon:
			finalTok := p.tok
			p.next()
			return finalTok
		case token.RightBrace:
			if p.scopeDepth > 0 {
				return p.prevTok
			}
		case token.EOF, token.Print, token.Var, token.If, token.While, token.For, token.Break, token.Continue, token.Return, token.Class, token.LeftBrace:
			return finalTok
		default:
		}
		finalTok = p.tok
		p.next()
	}
}

func (p *parser) parseDecl() (ast.Stmt, bool) {
	var stmt ast.Stmt
	ok := true
	switch tok := p.tok; {
	case p.match(token.Comment):
		stmt = p.parseComment(tok)
	case p.scopeDepth == p.classBodyScopeDepth && p.match(token.Ident, token.Static, token.Get, token.Set):
		stmt, ok = p.parseMethodDecl(tok)
	case p.match(token.Var):
		stmt, ok = p.parseVarDecl(tok)
	case p.tok.Type == token.Fun && p.nextTok.Type == token.Ident:
		p.match(token.Fun)
		stmt, ok = p.parseFunDecl(tok)
	case p.match(token.Class):
		stmt, ok = p.parseClassDecl(tok)
	default:
		stmt, ok = p.parseStmt()
	}
	if !ok {
		return stmt, false
	}

	if commentedStmt, ok := p.parseCommentedStmt(stmt); ok {
		return commentedStmt, ok
	}

	if p.parseComments && len(p.midStmtComments) > 0 {
		stmt = &ast.CommentedStmt{
			Stmt:    stmt,
			Comment: p.midStmtComments[0],
		}
		p.midStmtComments = p.midStmtComments[1:]
	}

	return stmt, true
}

func (p *parser) parseComment(commentTok token.Token) *ast.Comment {
	return &ast.Comment{Comment: commentTok}
}

func (p *parser) parseVarDecl(varTok token.Token) (*ast.VarDecl, bool) {
	decl := &ast.VarDecl{Var: varTok}
	var ok bool
	if decl.Name, ok = p.parseIdent("expected variable name"); !ok {
		return decl, false
	}
	if p.match(token.Equal) {
		if decl.Initialiser, ok = p.parseExpr(); !ok {
			return decl, false
		}
	}
	if decl.Semicolon, ok = p.expectSemicolon2(); !ok {
		return decl, false
	}
	return decl, true
}

func (p *parser) parseFunDecl(funTok token.Token) (*ast.FunDecl, bool) {
	decl := &ast.FunDecl{Fun: funTok}
	var ok bool
	if decl.Name, ok = p.parseIdent("expected function name"); !ok {
		return decl, false
	}
	if decl.Function, ok = p.parseFun(); !ok {
		return decl, false
	}
	return decl, true
}

func (p *parser) parseClassDecl(classTok token.Token) (*ast.ClassDecl, bool) {
	prevClassScopeDepth := p.classBodyScopeDepth
	p.classBodyScopeDepth = p.scopeDepth + 1
	defer func() { p.classBodyScopeDepth = prevClassScopeDepth }()

	decl := &ast.ClassDecl{Class: classTok}

	prevCurClassDecl := p.curClassDecl
	defer func() { p.curClassDecl = prevCurClassDecl }()
	p.curClassDecl = decl

	var ok bool

	if decl.Name, ok = p.parseIdent("expected class name"); !ok {
		return decl, false
	}

	if p.match(token.Less) {
		if decl.Superclass, ok = p.parseIdent("expected superclass name"); !ok {
			return decl, false
		}
	}

	leftBrace, ok := p.expect2(token.LeftBrace)
	if !ok {
		return decl, false
	}
	decl.Body, ok = p.parseBlock(leftBrace)
	for i, stmt := range slices.Backward(decl.Body.Stmts) {
		switch stmt.(type) {
		case *ast.MethodDecl, *ast.Comment:
		default:
			decl.Body.Stmts = slices.Delete(decl.Body.Stmts, i, i+1)
			p.addErrorf(classTok, "class body can only contain method declarations and comments")
		}
	}
	if !ok {
		return decl, false
	}

	return decl, true
}

func (p *parser) parseMethodDecl(firstTok token.Token) (*ast.MethodDecl, bool) {
	decl := &ast.MethodDecl{Class: p.curClassDecl}
	var ok bool
	if firstTok.Type == token.Ident {
		decl.Name = &ast.Ident{Token: firstTok}
	} else {
		decl.Modifiers = append(decl.Modifiers, firstTok)
		if firstTok.Type == token.Static {
			if tok, ok := p.match2(token.Get, token.Set); ok {
				decl.Modifiers = append(decl.Modifiers, tok)
			}
		}
		if decl.Name, ok = p.parseIdent("expected method name"); !ok {
			return decl, false
		}
	}
	if decl.Function, ok = p.parseFun(); !ok {
		return decl, false
	}
	return decl, true
}

func (p *parser) parseFun() (*ast.Function, bool) {
	fun := &ast.Function{}
	var ok bool
	if fun.LeftParen, ok = p.expect2(token.LeftParen); !ok {
		return nil, false
	}
	if !p.match(token.RightParen) {
		if fun.Params, ok = p.parseParams(); !ok {
			return fun, false
		}
		if !p.expect(token.RightParen) {
			return fun, false
		}
	}
	leftBrace, ok := p.expect2(token.LeftBrace)
	if !ok {
		return fun, false
	}
	if fun.Body, ok = p.parseBlock(leftBrace); !ok {
		return fun, false
	}
	return fun, true
}

func (p *parser) parseParams() ([]*ast.ParamDecl, bool) {
	var params []*ast.ParamDecl
	for {
		decl := &ast.ParamDecl{}
		var ok bool
		if decl.Name, ok = p.parseIdent("expected parameter name"); !ok {
			return params, false
		}
		params = append(params, decl)
		if !p.match(token.Comma) {
			break
		}
	}
	return params, true
}

func (p *parser) parseStmt() (ast.Stmt, bool) {
	var stmt ast.Stmt
	var ok bool
	switch tok := p.tok; {
	case p.match(token.Print):
		stmt, ok = p.parsePrintStmt(tok)
	case p.match(token.LeftBrace):
		stmt, ok = p.parseBlock(tok)
	case p.match(token.If):
		stmt, ok = p.parseIfStmt(tok)
	case p.match(token.While):
		stmt, ok = p.parseWhileStmt(tok)
	case p.match(token.For):
		stmt, ok = p.parseForStmt(tok)
	case p.match(token.Break):
		stmt, ok = p.parseBreakStmt(tok)
	case p.match(token.Continue):
		stmt, ok = p.parseContinueStmt(tok)
	case p.match(token.Return):
		stmt, ok = p.parseReturnStmt(tok)
	default:
		var exprStmt *ast.ExprStmt
		exprStmt, ok = p.parseExprStmt()
		// Avoid returning typed nil.
		if exprStmt != nil {
			stmt = exprStmt
		}
	}
	if !ok {
		return stmt, false
	}

	if commentedStmt, ok := p.parseCommentedStmt(stmt); ok {
		return commentedStmt, ok
	}

	return stmt, true
}

func (p *parser) parseCommentedStmt(stmt ast.Stmt) (*ast.CommentedStmt, bool) {
	comment, ok := p.matchFunc(func(tok token.Token) bool {
		return tok.Type == token.Comment && tok.Start().Line == stmt.End().Line
	})
	if ok && p.parseComments {
		return &ast.CommentedStmt{Stmt: stmt, Comment: p.parseComment(comment)}, true
	}
	return nil, false
}

func (p *parser) parseExprStmt() (*ast.ExprStmt, bool) {
	stmt := &ast.ExprStmt{}
	var ok bool
	if stmt.Expr, ok = p.parseExpr(); !ok {
		if stmt.Expr == nil {
			return nil, false
		}
		return stmt, false
	}
	if stmt.Semicolon, ok = p.expectSemicolon2(); !ok {
		return stmt, false
	}
	return stmt, true
}

func (p *parser) parsePrintStmt(printTok token.Token) (*ast.PrintStmt, bool) {
	stmt := &ast.PrintStmt{Print: printTok}
	var ok bool
	if stmt.Expr, ok = p.parseExpr(); !ok {
		return stmt, false
	}
	if stmt.Semicolon, ok = p.expectSemicolon2(); !ok {
		return stmt, false
	}
	return stmt, true
}

func (p *parser) parseBlock(leftBrace token.Token) (*ast.Block, bool) {
	p.scopeDepth++
	defer func() { p.scopeDepth-- }()
	decl := &ast.Block{LeftBrace: leftBrace}
	var ok bool
	decl.Stmts = p.parseDeclsUntil(token.RightBrace, token.EOF)
	if decl.RightBrace, ok = p.expect2(token.RightBrace); !ok {
		return decl, false
	}
	return decl, true
}

func (p *parser) parseIfStmt(ifTok token.Token) (*ast.IfStmt, bool) {
	stmt := &ast.IfStmt{If: ifTok}
	var ok bool
	if !p.expect(token.LeftParen) {
		return stmt, false
	}
	if stmt.Condition, ok = p.parseExpr(); !ok {
		return stmt, false
	}
	if !p.expect(token.RightParen) {
		return stmt, false
	}
	if stmt.Then, ok = p.parseStmt(); !ok {
		return stmt, false
	}
	if p.match(token.Else) {
		if stmt.Else, ok = p.parseStmt(); !ok {
			return stmt, false
		}
	}
	return stmt, true
}

func (p *parser) parseWhileStmt(whileTok token.Token) (*ast.WhileStmt, bool) {
	stmt := &ast.WhileStmt{While: whileTok}
	var ok bool
	if !p.expect(token.LeftParen) {
		return stmt, false
	}
	if stmt.Condition, ok = p.parseExpr(); !ok {
		return stmt, false
	}
	if !p.expect(token.RightParen) {
		return stmt, false
	}
	if stmt.Body, ok = p.parseStmt(); !ok {
		return stmt, false
	}
	return stmt, true
}

func (p *parser) parseForStmt(forTok token.Token) (*ast.ForStmt, bool) {
	stmt := &ast.ForStmt{For: forTok}
	var ok bool

	if !p.expect(token.LeftParen) {
		return stmt, false
	}
	switch tok := p.tok; {
	case p.match(token.Var):
		stmt.Initialise, ok = p.parseVarDecl(tok)
	case p.match(token.Semicolon):
		ok = true
	default:
		stmt.Initialise, ok = p.parseExprStmt()
	}
	if !ok {
		return stmt, false
	}

	if !p.match(token.Semicolon) {
		if stmt.Condition, ok = p.parseExpr(); !ok {
			return stmt, false
		}
		if !p.expectSemicolon() {
			return stmt, false
		}
	}

	if !p.match(token.RightParen) {
		if stmt.Update, ok = p.parseExpr(); !ok {
			return stmt, false
		}
		if !p.expect(token.RightParen) {
			return stmt, false
		}
	}

	if stmt.Body, ok = p.parseStmt(); !ok {
		return stmt, false
	}

	return stmt, true
}

func (p *parser) parseBreakStmt(breakTok token.Token) (*ast.BreakStmt, bool) {
	stmt := &ast.BreakStmt{Break: breakTok}
	var ok bool
	if stmt.Semicolon, ok = p.expectSemicolon2(); !ok {
		return stmt, false
	}
	return stmt, true
}

func (p *parser) parseContinueStmt(continueTok token.Token) (*ast.ContinueStmt, bool) {
	stmt := &ast.ContinueStmt{Continue: continueTok}
	var ok bool
	if stmt.Semicolon, ok = p.expectSemicolon2(); !ok {
		return stmt, false
	}
	return stmt, true
}

func (p *parser) parseReturnStmt(returnTok token.Token) (*ast.ReturnStmt, bool) {
	stmt := &ast.ReturnStmt{Return: returnTok}
	var ok bool
	if stmt.Semicolon, ok = p.match2(token.Semicolon); !ok {
		if stmt.Value, ok = p.parseExpr(); !ok {
			return stmt, false
		}
		if stmt.Semicolon, ok = p.expectSemicolon2(); !ok {
			return stmt, false
		}
	}
	return stmt, true
}

func (p *parser) parseExpr() (ast.Expr, bool) {
	return p.parseCommaExpr()
}

func (p *parser) parseCommaExpr() (ast.Expr, bool) {
	if p.extraFeatures {
		return p.parseBinaryExpr(p.parseAssignmentExpr, token.Comma)
	} else {
		return p.parseAssignmentExpr()
	}
}

func (p *parser) parseAssignmentExpr() (ast.Expr, bool) {
	var expr ast.Expr
	var ok bool
	if expr, ok = p.parseTernaryExpr(); !ok {
		return expr, false
	}
	if p.match(token.Equal) {
		switch left := expr.(type) {
		case *ast.IdentExpr:
			assignmentExpr := &ast.AssignmentExpr{Left: left.Ident}
			expr = assignmentExpr
			if assignmentExpr.Right, ok = p.parseAssignmentExpr(); !ok {
				return expr, false
			}
		case *ast.GetExpr:
			setExpr := &ast.SetExpr{Object: left.Object, Name: left.Name}
			expr = setExpr
			if setExpr.Value, ok = p.parseAssignmentExpr(); !ok {
				return expr, false
			}
		default:
			p.addErrorf(expr, "invalid assignment target")
		}
	}
	return expr, true
}

func (p *parser) parseTernaryExpr() (ast.Expr, bool) {
	var expr ast.Expr
	var ok bool
	if expr, ok = p.parseLogicalOrExpr(); !ok {
		return expr, false
	}
	if p.match(token.Question) {
		ternaryExpr := &ast.TernaryExpr{Condition: expr}
		expr = ternaryExpr
		if ternaryExpr.Then, ok = p.parseExpr(); !ok {
			return expr, false
		}
		if !p.expect(token.Colon) {
			return expr, false
		}
		if ternaryExpr.Else, ok = p.parseTernaryExpr(); !ok {
			return expr, false
		}
	}
	return expr, true
}

func (p *parser) parseLogicalOrExpr() (ast.Expr, bool) {
	return p.parseBinaryExpr(p.parseLogicalAndExpr, token.Or)
}

func (p *parser) parseLogicalAndExpr() (ast.Expr, bool) {
	return p.parseBinaryExpr(p.parseEqualityExpr, token.And)
}

func (p *parser) parseEqualityExpr() (ast.Expr, bool) {
	return p.parseBinaryExpr(p.parseRelationalExpr, token.EqualEqual, token.BangEqual)
}

func (p *parser) parseRelationalExpr() (ast.Expr, bool) {
	return p.parseBinaryExpr(p.parseAdditiveExpr, token.Less, token.LessEqual, token.Greater, token.GreaterEqual)
}

func (p *parser) parseAdditiveExpr() (ast.Expr, bool) {
	return p.parseBinaryExpr(p.parseMultiplicativeExpr, token.Plus, token.Minus)
}

func (p *parser) parseMultiplicativeExpr() (ast.Expr, bool) {
	return p.parseBinaryExpr(p.parseUnaryExpr, token.Asterisk, token.Slash, token.Percent)
}

// parseBinaryExpr parses a binary expression which uses the given operators. next is a function which parses an
// expression of next highest precedence.
func (p *parser) parseBinaryExpr(next func() (ast.Expr, bool), operators ...token.Type) (ast.Expr, bool) {
	var expr ast.Expr
	var ok bool
	if expr, ok = next(); !ok {
		return expr, false
	}
	for {
		binaryExpr := &ast.BinaryExpr{Left: expr}
		if binaryExpr.Op, ok = p.match2(operators...); !ok {
			break
		}
		expr = binaryExpr
		if binaryExpr.Right, ok = next(); !ok {
			return expr, false
		}
	}
	return expr, true
}

func (p *parser) parseUnaryExpr() (ast.Expr, bool) {
	if op, ok := p.match2(token.Bang, token.Minus); ok {
		expr := &ast.UnaryExpr{Op: op}
		if expr.Right, ok = p.parseUnaryExpr(); !ok {
			return expr, false
		}
		return expr, true
	}
	return p.parseCallExpr()
}

func (p *parser) parseCallExpr() (ast.Expr, bool) {
	var expr ast.Expr
	var ok bool
	if expr, ok = p.parsePrimaryExpr(); !ok {
		return expr, false
	}
	for {
		switch tok := p.tok; {
		case p.match(token.LeftParen):
			callExpr := &ast.CallExpr{LeftParen: tok, Callee: expr}
			expr = callExpr
			if callExpr.RightParen, ok = p.match2(token.RightParen); !ok {
				if callExpr.Args, callExpr.Commas, ok = p.parseArgs(); !ok {
					return expr, false
				}
				if callExpr.RightParen, ok = p.expect2(token.RightParen); !ok {
					return expr, false
				}
			}
		case p.match(token.Dot):
			getExpr := &ast.GetExpr{Object: expr, Dot: tok}
			expr = getExpr
			if getExpr.Name, ok = p.parseIdent("expected property name"); !ok {
				return expr, false
			}
		default:
			return expr, true
		}
	}
}

func (p *parser) parseArgs() ([]ast.Expr, []token.Token, bool) {
	var args []ast.Expr
	var commas []token.Token
	for {
		arg, ok := p.parseAssignmentExpr()
		if arg != nil {
			args = append(args, arg)
		}
		if !ok {
			return args, commas, false
		}
		comma, ok := p.match2(token.Comma)
		if !ok {
			break
		}
		commas = append(commas, comma)
	}
	return args, commas, true
}

func (p *parser) parsePrimaryExpr() (ast.Expr, bool) {
	switch tok := p.tok; {
	case p.match(token.Number, token.String, token.True, token.False, token.Nil):
		return &ast.LiteralExpr{Value: tok}, true
	case p.match(token.Ident):
		return &ast.IdentExpr{Ident: &ast.Ident{Token: tok}}, true
	case p.match(token.This):
		return &ast.ThisExpr{This: tok}, true
	case p.match(token.Super):
		superExpr := &ast.SuperExpr{Super: tok}
		getExpr := &ast.GetExpr{Object: superExpr}
		var ok bool
		if getExpr.Dot, ok = p.expect2(token.Dot); !ok {
			return superExpr, false
		}
		if getExpr.Name, ok = p.parseIdent("expected property name"); !ok {
			return getExpr, false
		}
		return getExpr, true
	case p.extraFeatures && p.match(token.Fun):
		return p.parseFunExpr(tok)
	case p.match(token.LeftParen):
		expr := &ast.GroupExpr{LeftParen: tok}
		var ok bool
		if expr.Expr, ok = p.parseExpr(); !ok {
			return expr, false
		}
		if expr.RightParen, ok = p.expect2(token.RightParen); !ok {
			return expr, false
		}
		return expr, true
	case p.match(token.Comment):
		p.midStmtComments = append(p.midStmtComments, p.parseComment(tok))
		return p.parsePrimaryExpr()
	// Error productions
	case p.match(token.EqualEqual, token.BangEqual, token.Less, token.LessEqual, token.Greater, token.GreaterEqual, token.Asterisk, token.Slash, token.Plus):
		p.addErrorf(tok, "binary operator %m must have left and right operands", tok.Type)
		expr := &ast.BinaryExpr{Op: tok}
		var parseExpr func() (ast.Expr, bool)
		switch tok.Type {
		case token.EqualEqual, token.BangEqual:
			parseExpr = p.parseEqualityExpr
		case token.Less, token.LessEqual, token.Greater, token.GreaterEqual:
			parseExpr = p.parseRelationalExpr
		case token.Plus:
			parseExpr = p.parseAdditiveExpr
		case token.Asterisk, token.Slash:
			parseExpr = p.parseMultiplicativeExpr
		default:
		}
		var ok bool
		if expr.Right, ok = parseExpr(); !ok {
			return expr, false
		}
		return expr, true
	default:
		p.addErrorf(tok, "expected expression")
		return nil, false
	}
}

func (p *parser) parseFunExpr(funTok token.Token) (*ast.FunExpr, bool) {
	expr := &ast.FunExpr{Fun: funTok}
	var ok bool
	if expr.Function, ok = p.parseFun(); !ok {
		return expr, false
	}
	return expr, true
}

func (p *parser) parseIdent(errMsg string) (*ast.Ident, bool) {
	name, ok := p.expect2f(token.Ident, "%s", errMsg)
	if !ok {
		return nil, false
	}
	return &ast.Ident{Token: name}, true
}

// match reports whether the current token is one of the given types and advances the parser if so.
func (p *parser) match(types ...token.Type) bool {
	if slices.Contains(types, p.tok.Type) {
		p.next()
		return true
	}
	return false
}

// match2 is like match but also returns the matched token.
func (p *parser) match2(types ...token.Type) (token.Token, bool) {
	tok := p.tok
	if p.match(types...) {
		return tok, true
	}
	return token.Token{}, false
}

// matchFunc reports whether the current token satisfies the given predicate and advances the parser if so.
func (p *parser) matchFunc(f func(token.Token) bool) (token.Token, bool) {
	tok := p.tok
	if f(tok) {
		p.next()
		return tok, true
	}
	return token.Token{}, false
}

// expect reports whether the current token is of the given type. If it is, the parser is advanced. Otherwise, an
// "expected %m" error is added.
func (p *parser) expect(t token.Type) bool {
	_, ok := p.expect2(t)
	return ok
}

// expect2 is like expect but also returns the matched token.
func (p *parser) expect2(t token.Type) (token.Token, bool) {
	return p.expect2f(t, "expected %m", t)
}

// expect2f is like expect2 but accepts a format string for the error message.
func (p *parser) expect2f(t token.Type, format string, a ...any) (token.Token, bool) {
	tok, ok := p.match2(t)
	if !ok {
		p.addErrorf(p.tok, format, a...)
		return token.Token{}, false
	}
	return tok, true
}

// expectSemicolon reports whether the current token is a semicolon. If it is, the parser is advanced. Otherwise, an
// "expected trailing ;" error is added.
func (p *parser) expectSemicolon() bool {
	_, ok := p.expectSemicolon2()
	return ok
}

// expectSemicolon2 is like expectSemicolon but also returns the matched token.
func (p *parser) expectSemicolon2() (token.Token, bool) {
	return p.expect2f(token.Semicolon, "expected trailing %m", token.Semicolon)
}

// next advances the parser to the next token.
func (p *parser) next() {
	p.prevTok = p.tok
	p.tok = p.nextTok
	p.nextTok = p.lexer.Next()
	// next may be called after EOF has been reached but we don't want to print it out multiple times
	if p.printTokens && p.tok.Type != token.EOF {
		fmt.Println(p.nextTok)
	}
	if p.tok.Type == token.Comment && !p.parseComments {
		p.next()
	}
}

func (p *parser) addErrorf(rang token.Range, format string, args ...any) {
	start := rang.Start()
	if len(p.errs) > 0 && start == p.lastErrPos {
		return
	}
	p.lastErrPos = start
	p.errs.Addf(rang, loxerr.Fatal, format, args...)
}
