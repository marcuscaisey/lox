package analysis

import (
	"github.com/marcuscaisey/lox/lox"
	"github.com/marcuscaisey/lox/lox/ast"
	"github.com/marcuscaisey/lox/lox/token"
)

const (
	maxParams = 255
	maxArgs   = maxParams
)

// CheckSemantics checks that the following rules have been followed:
//   - Write-only properties are not allowed
//   - break and continue can only be used inside a loop
//   - return can only be used inside a function definition
//   - init() cannot return a value
//   - _ cannot be used in a non-assignment expression
//   - this can only be used inside a method definition
//   - property getter cannot have parameters
//   - property setter must have exactly one parameter
//   - functions cannot have more than 255 parameters
//   - function calls cannot have more than 255 arguments
func CheckSemantics(program ast.Program) lox.Errors {
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
	ast.Walk(program, c.walk)
	return c.errs
}

func (c *semanticChecker) walk(node ast.Node) bool {
	switch node := node.(type) {
	case ast.FunDecl:
		c.walkFun(node.Function, funTypeFunction)
		return false
	case ast.ClassDecl:
		c.checkNoWriteOnlyProperties(node.Methods())
	case ast.MethodDecl:
		c.checkNumPropertyParams(node)
		c.walkFun(node.Function, methodFunType(node))
		return false
	case ast.WhileStmt:
		c.walkWhileStmt(node)
		return false
	case ast.ForStmt:
		c.walkForStmt(node)
		return false
	case ast.BreakStmt:
		c.checkBreakInLoop(node)
	case ast.ContinueStmt:
		c.checkContinueInLoop(node)
	case ast.ReturnStmt:
		c.checkReturnInFun(node)
		c.checkNoConstructorReturn(node)
	case ast.FunExpr:
		c.walkFun(node.Function, funTypeFunction)
		return false
	case ast.VariableExpr:
		c.checkNoPlaceholderAssignment(node)
	case ast.ThisExpr:
		c.checkThisInMethod(node)
	case ast.CallExpr:
		c.checkNumArgs(node.Args)
	default:
	}
	return true
}

func (c *semanticChecker) walkFun(fun ast.Function, funType funType) {
	c.checkNumParams(fun.Params)

	// Break and continue are not allowed to jump out of a function so reset the loop depth to catch any invalid uses.
	prevInLoop := c.inLoop
	c.inLoop = false
	defer func() { c.inLoop = prevInLoop }()

	prevFunType := c.curFunType
	c.curFunType = funType
	defer func() { c.curFunType = prevFunType }()

	for _, stmt := range fun.Body.Stmts {
		ast.Walk(stmt, c.walk)
	}
}

func (c *semanticChecker) checkNumParams(params []token.Token) {
	if len(params) > maxParams {
		c.errs.Addf(params[maxParams], "cannot define more than %d function parameters", maxParams)
	}
}

func (c *semanticChecker) walkWhileStmt(stmt ast.WhileStmt) {
	ast.Walk(stmt.Condition, c.walk)
	endLoop := c.beginLoop()
	defer endLoop()
	ast.Walk(stmt.Body, c.walk)
}

func (c *semanticChecker) walkForStmt(stmt ast.ForStmt) {
	if stmt.Initialise != nil {
		ast.Walk(stmt.Initialise, c.walk)
	}
	if stmt.Condition != nil {
		ast.Walk(stmt.Condition, c.walk)
	}
	if stmt.Update != nil {
		ast.Walk(stmt.Update, c.walk)
	}
	endLoop := c.beginLoop()
	defer endLoop()
	ast.Walk(stmt.Body, c.walk)
}

// beginLoop sets the inLoop flag to true and returns a function which resets it to its previous value
func (c *semanticChecker) beginLoop() func() {
	prev := c.inLoop
	c.inLoop = true
	return func() { c.inLoop = prev }
}

func (c *semanticChecker) checkNoWriteOnlyProperties(methods []ast.MethodDecl) {
	gettersByName := map[string]bool{}
	setterNameToksByName := map[string]token.Token{}
	for _, methodDecl := range methods {
		switch {
		case methodDecl.HasModifier(token.Get):
			gettersByName[methodDecl.Name.Lexeme] = true
		case methodDecl.HasModifier(token.Set):
			setterNameToksByName[methodDecl.Name.Lexeme] = methodDecl.Name
		}
	}
	for name, nameTok := range setterNameToksByName {
		if !gettersByName[name] {
			c.errs.Add(nameTok, "write-only properties are not allowed")
		}
	}
}

func (c *semanticChecker) checkNumPropertyParams(decl ast.MethodDecl) {
	switch {
	case decl.HasModifier(token.Get) && len(decl.Params) > 0:
		c.errs.Add(decl.Params[0:], "property getter cannot have parameters")
	case decl.HasModifier(token.Set):
		if len(decl.Params) == 0 {
			c.errs.Add(decl.Name, "property setter must have a parameter")
		} else if len(decl.Params) > 1 {
			c.errs.Add(decl.Params[1:], "property setter can only have one parameter")
		}
	}

}

func (c *semanticChecker) checkBreakInLoop(stmt ast.BreakStmt) {
	if !c.inLoop {
		c.errs.Addf(stmt, "%m can only be used inside a loop", token.Break)
	}
}

func (c *semanticChecker) checkContinueInLoop(stmt ast.ContinueStmt) {
	if !c.inLoop {
		c.errs.Addf(stmt, "%m can only be used inside a loop", token.Continue)
	}
}

func (c *semanticChecker) checkReturnInFun(stmt ast.ReturnStmt) {
	if c.curFunType == funTypeNone {
		c.errs.Addf(stmt, "%m can only be used inside a function definition", token.Return)
	}
}

func (c *semanticChecker) checkNoConstructorReturn(stmt ast.ReturnStmt) {
	if stmt.Value != nil && c.curFunType.IsConstructor() {
		c.errs.Addf(stmt, "%s() cannot return a value", token.ConstructorIdent)
	}
}

func (c *semanticChecker) checkNoPlaceholderAssignment(expr ast.VariableExpr) {
	if expr.Name.Lexeme == token.PlaceholderIdent {
		c.errs.Addf(expr.Name, "identifier %s cannot be used in a non-assignment expression", token.PlaceholderIdent)
	}
}

func (c *semanticChecker) checkThisInMethod(expr ast.ThisExpr) {
	if !c.curFunType.IsMethod() {
		c.errs.Addf(expr, "%m can only be used inside a method definition", token.This)
	}
}

func (c *semanticChecker) checkNumArgs(args []ast.Expr) {
	if len(args) > maxArgs {
		c.errs.Addf(args[maxArgs], "cannot pass more than %d arguments to function", maxArgs)
	}
}

type funType int

const (
	funTypeNone     funType = iota
	funTypeFunction funType = 1 << (iota - 1)
	funTypeMethodFlag
	funTypeConstructorFlag
)

func (f funType) IsMethod() bool {
	return f&funTypeMethodFlag != 0
}

func (f funType) IsConstructor() bool {
	return f&funTypeConstructorFlag != 0
}

func methodFunType(decl ast.MethodDecl) funType {
	typ := funTypeFunction | funTypeMethodFlag
	if decl.IsConstructor() {
		typ |= funTypeConstructorFlag
	}
	return typ
}
