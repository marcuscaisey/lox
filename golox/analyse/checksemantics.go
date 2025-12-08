package analyse

import (
	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerr"
	"github.com/marcuscaisey/lox/golox/token"
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
//   - _ cannot be used as a value
//   - _ cannot be used as a field name
//   - this can only be used inside a method definition
//   - property getter cannot have parameters
//   - property setter must have exactly one parameter
//   - functions cannot have more than 255 parameters
//   - function calls cannot have more than 255 arguments
//
// If there is an error, it will be of type [loxerr.Errors].
func CheckSemantics(program *ast.Program, opts ...Option) error {
	cfg := newConfig(opts)
	c := &semanticChecker{extraFeatures: cfg.extraFeatures}
	return c.Check(program)
}

type semanticChecker struct {
	extraFeatures bool

	inLoop     bool
	curFunType funType
	inMethod   bool

	errs loxerr.Errors
}

func (c *semanticChecker) Check(program *ast.Program) error {
	ast.Walk(program, c.walk)
	return c.errs.Err()
}

func (c *semanticChecker) walk(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.FunDecl:
		c.walkFun(node.Function, funTypeFunction)
		return false
	case *ast.ClassDecl:
		c.checkNoWriteOnlyProperties(node.Methods())
	case *ast.MethodDecl:
		c.checkNumPropertyParams(node)
		c.walkFun(node.Function, methodFunType(node))
		return false
	case *ast.WhileStmt:
		c.walkWhileStmt(node)
		return false
	case *ast.ForStmt:
		c.walkForStmt(node)
		return false
	case *ast.BreakStmt:
		c.checkBreakInLoop(node)
	case *ast.ContinueStmt:
		c.checkContinueInLoop(node)
	case *ast.ReturnStmt:
		c.checkReturnInFun(node)
		c.checkNoConstructorReturn(node)
	case *ast.FunExpr:
		c.walkFun(node.Function, funTypeFunction)
		return false
	case *ast.IdentExpr:
		c.checkNoPlaceholderAccess(node)
	case *ast.ThisExpr:
		c.checkThisInMethod(node)
	case *ast.CallExpr:
		c.checkNumArgs(node.Args)
	case *ast.GetExpr:
		c.checkNoPlaceholderFieldAccess(node.Name)
	case *ast.SetExpr:
		c.checkNoPlaceholderFieldAccess(node.Name)
	default:
	}
	return true
}

func (c *semanticChecker) walkFun(fun *ast.Function, funType funType) {
	if fun == nil {
		return
	}

	c.checkNumParams(fun.Params)

	// Break and continue are not allowed to jump out of a function so reset the loop depth to catch any invalid uses.
	prevInLoop := c.inLoop
	c.inLoop = false
	defer func() { c.inLoop = prevInLoop }()

	prevFunType := c.curFunType
	c.curFunType = funType
	defer func() { c.curFunType = prevFunType }()

	if funType.IsMethod() {
		prevInMethod := c.inMethod
		c.inMethod = true
		defer func() { c.inMethod = prevInMethod }()
	}

	ast.Walk(fun.Body, c.walk)
}

func (c *semanticChecker) checkNumParams(params []*ast.ParamDecl) {
	if len(params) > maxParams {
		c.errs.Addf(params[maxParams], loxerr.Fatal, "cannot define more than %d function parameters", maxParams)
	}
}

func (c *semanticChecker) walkWhileStmt(stmt *ast.WhileStmt) {
	ast.Walk(stmt.Condition, c.walk)
	endLoop := c.beginLoop()
	defer endLoop()
	ast.Walk(stmt.Body, c.walk)
}

func (c *semanticChecker) walkForStmt(stmt *ast.ForStmt) {
	ast.Walk(stmt.Initialise, c.walk)
	ast.Walk(stmt.Condition, c.walk)
	ast.Walk(stmt.Update, c.walk)
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

func (c *semanticChecker) checkNoWriteOnlyProperties(methods []*ast.MethodDecl) {
	gettersByName := map[string]bool{}
	setterIdentsByName := map[string]*ast.Ident{}
	for _, methodDecl := range methods {
		if !methodDecl.Name.IsValid() {
			continue
		}
		switch {
		case methodDecl.HasModifier(token.Get):
			gettersByName[methodDecl.Name.String()] = true
		case methodDecl.HasModifier(token.Set):
			setterIdentsByName[methodDecl.Name.String()] = methodDecl.Name
		}
	}
	for name, ident := range setterIdentsByName {
		if !gettersByName[name] {
			c.errs.Addf(ident, loxerr.Fatal, "write-only properties are not allowed")
		}
	}
}

func (c *semanticChecker) checkNumPropertyParams(decl *ast.MethodDecl) {
	if decl.Function == nil {
		return
	}
	switch {
	case decl.HasModifier(token.Get) && len(decl.Function.Params) > 0:
		c.errs.AddSpanningRangesf(decl.Function.Params[0], decl.Function.Params[len(decl.Function.Params)-1], loxerr.Fatal, "property getter cannot have parameters")
	case decl.HasModifier(token.Set):
		if len(decl.Function.Params) == 0 {
			c.errs.Addf(decl.Name, loxerr.Fatal, "property setter must have a parameter")
		} else if len(decl.Function.Params) > 1 {
			c.errs.AddSpanningRangesf(decl.Function.Params[1], decl.Function.Params[len(decl.Function.Params)-1], loxerr.Fatal, "property setter can only have one parameter")
		}
	}

}

func (c *semanticChecker) checkBreakInLoop(stmt *ast.BreakStmt) {
	if !c.inLoop {
		c.errs.Addf(stmt, loxerr.Fatal, "%m can only be used inside a loop", token.Break)
	}
}

func (c *semanticChecker) checkContinueInLoop(stmt *ast.ContinueStmt) {
	if !c.inLoop {
		c.errs.Addf(stmt, loxerr.Fatal, "%m can only be used inside a loop", token.Continue)
	}
}

func (c *semanticChecker) checkReturnInFun(stmt *ast.ReturnStmt) {
	if c.curFunType == funTypeNone {
		c.errs.Addf(stmt, loxerr.Fatal, "%m can only be used inside a function definition", token.Return)
	}
}

func (c *semanticChecker) checkNoConstructorReturn(stmt *ast.ReturnStmt) {
	if stmt.Value != nil && c.curFunType.IsConstructor() {
		c.errs.Addf(stmt, loxerr.Fatal, "%s() cannot return a value", token.ConstructorIdent)
	}
}

func (c *semanticChecker) checkNoPlaceholderAccess(expr *ast.IdentExpr) {
	if c.extraFeatures && expr.Ident.String() == token.PlaceholderIdent {
		c.errs.Addf(expr.Ident, loxerr.Fatal, "%s cannot be used as a value", token.PlaceholderIdent)
	}
}

func (c *semanticChecker) checkNoPlaceholderFieldAccess(ident *ast.Ident) {
	if c.extraFeatures && ident.IsValid() && ident.String() == token.PlaceholderIdent {
		c.errs.Addf(ident, loxerr.Fatal, "%s cannot be used as a field name", token.PlaceholderIdent)
	}
}

func (c *semanticChecker) checkThisInMethod(expr *ast.ThisExpr) {
	if !c.inMethod {
		c.errs.Addf(expr, loxerr.Fatal, "%m can only be used inside a method definition", token.This)
	}
}

func (c *semanticChecker) checkNumArgs(args []ast.Expr) {
	if len(args) > maxArgs {
		c.errs.Addf(args[maxArgs], loxerr.Fatal, "cannot pass more than %d arguments to function", maxArgs)
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

func methodFunType(decl *ast.MethodDecl) funType {
	typ := funTypeFunction | funTypeMethodFlag
	if decl.IsConstructor() {
		typ |= funTypeConstructorFlag
	}
	return typ
}
