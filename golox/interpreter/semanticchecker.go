package interpreter

import (
	"fmt"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/lox"
	"github.com/marcuscaisey/lox/golox/token"
)

// checkSemantics checks that the following rules have been followed:
//   - Write-only properties are not allowed
//   - break and continue can only be used inside a loop
//   - return can only be used inside a function definition
//   - init() cannot return a value
//   - _ cannot be used in a non-assignment expression
//   - this can only be used inside a method definition
func checkSemantics(program ast.Program) lox.Errors {
	c := newSemanticChecker()
	return c.Check(program)
}

type semanticChecker struct {
	inLoop     bool
	curFunType funType

	errs lox.Errors
}

func newSemanticChecker() *semanticChecker {
	return &semanticChecker{}
}

func (c *semanticChecker) Check(program ast.Program) lox.Errors {
	c.checkProgram(program)
	return c.errs
}

func (c *semanticChecker) checkProgram(program ast.Program) {
	for _, stmt := range program.Stmts {
		c.checkStmt(stmt)
	}
}

func (c *semanticChecker) checkStmt(stmt ast.Stmt) {
	switch stmt := stmt.(type) {
	case ast.VarDecl:
		c.checkVarDecl(stmt)
	case ast.FunDecl:
		c.checkFunDecl(stmt)
	case ast.ClassDecl:
		c.checkClassDecl(stmt)
	case ast.ExprStmt:
		c.checkExprStmt(stmt)
	case ast.PrintStmt:
		c.checkPrintStmt(stmt)
	case ast.BlockStmt:
		c.checkBlockStmt(stmt)
	case ast.IfStmt:
		c.checkIfStmt(stmt)
	case ast.WhileStmt:
		c.checkWhileStmt(stmt)
	case ast.ForStmt:
		c.checkForStmt(stmt)
	case ast.BreakStmt:
		c.checkBreakStmt(stmt)
	case ast.ContinueStmt:
		c.checkContinueStmt(stmt)
	case ast.ReturnStmt:
		c.checkReturnStmt(stmt)
	default:
		panic(fmt.Sprintf("unexpected statement type: %T", stmt))
	}
}

func (c *semanticChecker) checkVarDecl(stmt ast.VarDecl) {
	if stmt.Initialiser != nil {
		c.checkExpr(stmt.Initialiser)
	}
}

func (c *semanticChecker) checkFunDecl(stmt ast.FunDecl) {
	c.checkFun(stmt.Params, stmt.Body, funTypeFunction)
}

func (c *semanticChecker) checkFun(_ []token.Token, body []ast.Stmt, funType funType) {
	// Break and continue are not allowed to jump out of a function so reset the loop depth to catch any invalid uses.
	prevInLoop := c.inLoop
	c.inLoop = false
	defer func() { c.inLoop = prevInLoop }()

	prevFunType := c.curFunType
	c.curFunType = funType
	defer func() { c.curFunType = prevFunType }()

	for _, stmt := range body {
		c.checkStmt(stmt)
	}
}

func (c *semanticChecker) checkClassDecl(stmt ast.ClassDecl) {
	for _, methodDecl := range stmt.Methods {
		c.checkFun(methodDecl.Params, methodDecl.Body, methodFunType(methodDecl))
	}
	c.checkNoWriteOnlyProperties(stmt)
}

func (c *semanticChecker) checkNoWriteOnlyProperties(stmt ast.ClassDecl) {
	gettersByName := map[string]bool{}
	setterNameToksByName := map[string]token.Token{}
	for _, methodDecl := range stmt.Methods {
		switch {
		case methodDecl.HasModifier(token.Get):
			gettersByName[methodDecl.Name.Lexeme] = true
		case methodDecl.HasModifier(token.Set):
			setterNameToksByName[methodDecl.Name.Lexeme] = methodDecl.Name
		}
	}
	for name, nameTok := range setterNameToksByName {
		if !gettersByName[name] {
			c.errs.Add(lox.FromToken(nameTok), "write-only properties are not allowed")
		}
	}
}

func (c *semanticChecker) checkExprStmt(stmt ast.ExprStmt) {
	c.checkExpr(stmt.Expr)
}

func (c *semanticChecker) checkPrintStmt(stmt ast.PrintStmt) {
	c.checkExpr(stmt.Expr)
}

func (c *semanticChecker) checkBlockStmt(stmt ast.BlockStmt) {
	for _, stmt := range stmt.Stmts {
		c.checkStmt(stmt)
	}
}

func (c *semanticChecker) checkIfStmt(stmt ast.IfStmt) {
	c.checkExpr(stmt.Condition)
	c.checkStmt(stmt.Then)
	if stmt.Else != nil {
		c.checkStmt(stmt.Else)
	}
}

func (c *semanticChecker) checkWhileStmt(stmt ast.WhileStmt) {
	c.checkExpr(stmt.Condition)
	endLoop := c.beginLoop()
	defer endLoop()
	c.checkStmt(stmt.Body)
}

func (c *semanticChecker) checkForStmt(stmt ast.ForStmt) {
	if stmt.Initialise != nil {
		c.checkStmt(stmt.Initialise)
	}
	if stmt.Condition != nil {
		c.checkExpr(stmt.Condition)
	}
	if stmt.Update != nil {
		c.checkExpr(stmt.Update)
	}
	endLoop := c.beginLoop()
	defer endLoop()
	c.checkStmt(stmt.Body)
}

// beginLoop sets the inLoop flag to true and returns a function which resets it to its previous value
func (c *semanticChecker) beginLoop() func() {
	prev := c.inLoop
	c.inLoop = true
	return func() { c.inLoop = prev }
}

func (c *semanticChecker) checkBreakStmt(stmt ast.BreakStmt) {
	if !c.inLoop {
		c.errs.Addf(lox.FromNode(stmt), "%m can only be used inside a loop", token.Break)
	}
}

func (c *semanticChecker) checkContinueStmt(stmt ast.ContinueStmt) {
	if !c.inLoop {
		c.errs.Addf(lox.FromNode(stmt), "%m can only be used inside a loop", token.Continue)
	}
}

func (c *semanticChecker) checkReturnStmt(stmt ast.ReturnStmt) {
	if c.curFunType == funTypeNone {
		c.errs.Addf(lox.FromNode(stmt), "%m can only be used inside a function definition", token.Return)
	}
	if stmt.Value != nil {
		if c.curFunType.IsConstructor() {
			c.errs.Addf(lox.FromNode(stmt), "%s() cannot return a value", token.ConstructorIdent)
		}
		c.checkExpr(stmt.Value)
	}
}

func (c *semanticChecker) checkExpr(expr ast.Expr) {
	switch expr := expr.(type) {
	case ast.FunExpr:
		c.checkFunExpr(expr)
	case ast.GroupExpr:
		c.checkGroupExpr(expr)
	case ast.LiteralExpr:
		// Nothing to check
	case ast.VariableExpr:
		c.checkVariableExpr(expr)
	case ast.ThisExpr:
		c.checkThisExpr(expr)
	case ast.CallExpr:
		c.checkCallExpr(expr)
	case ast.GetExpr:
		c.checkGetExpr(expr)
	case ast.UnaryExpr:
		c.checkUnaryExpr(expr)
	case ast.BinaryExpr:
		c.checkBinaryExpr(expr)
	case ast.TernaryExpr:
		c.checkTernaryExpr(expr)
	case ast.AssignmentExpr:
		c.checkAssignmentExpr(expr)
	case ast.SetExpr:
		c.checkSetExpr(expr)
	default:
		panic(fmt.Sprintf("unexpected expression type: %T", expr))
	}
}

func (c *semanticChecker) checkFunExpr(expr ast.FunExpr) {
	c.checkFun(expr.Params, expr.Body, funTypeFunction)
}

func (c *semanticChecker) checkGroupExpr(expr ast.GroupExpr) {
	c.checkExpr(expr.Expr)
}

func (c *semanticChecker) checkVariableExpr(expr ast.VariableExpr) {
	if expr.Name.Lexeme == token.PlaceholderIdent {
		c.errs.Addf(lox.FromToken(expr.Name), "identifier %s cannot be used in a non-assignment expression", token.PlaceholderIdent)
	}
}

func (c *semanticChecker) checkThisExpr(expr ast.ThisExpr) {
	if !c.curFunType.IsMethod() {
		c.errs.Addf(lox.FromNode(expr), "%m can only be used inside a method definition", token.This)
	}
}

func (c *semanticChecker) checkBinaryExpr(expr ast.BinaryExpr) {
	c.checkExpr(expr.Left)
	c.checkExpr(expr.Right)
}

func (c *semanticChecker) checkTernaryExpr(expr ast.TernaryExpr) {
	c.checkExpr(expr.Condition)
	c.checkExpr(expr.Then)
	c.checkExpr(expr.Else)
}

func (c *semanticChecker) checkCallExpr(expr ast.CallExpr) {
	c.checkExpr(expr.Callee)
	for _, arg := range expr.Args {
		c.checkExpr(arg)
	}
}

func (c *semanticChecker) checkGetExpr(expr ast.GetExpr) {
	c.checkExpr(expr.Object)
}

func (c *semanticChecker) checkUnaryExpr(expr ast.UnaryExpr) {
	c.checkExpr(expr.Right)
}

func (c *semanticChecker) checkAssignmentExpr(expr ast.AssignmentExpr) {
	c.checkExpr(expr.Right)
}

func (c *semanticChecker) checkSetExpr(expr ast.SetExpr) {
	c.checkExpr(expr.Value)
	c.checkExpr(expr.Object)
}
