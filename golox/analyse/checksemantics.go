package analyse

import (
	"fmt"
	"strings"

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
//   - init() cannot be static
//   - _ cannot be used as a value
//   - _ cannot be used as a field name
//   - this can only be used inside a method definition
//   - super can only be used inside a method definition
//   - super can only be used inside a subclass
//   - super properties cannot be assigned to
//   - property getter cannot have parameters
//   - property setter must have exactly one parameter
//   - functions cannot have more than 255 parameters
//   - function calls cannot have more than 255 arguments
//   - classes cannot inherit from themselves
//   - classes cannot have two methods with the same name and modifiers
//   - classes cannot have a property accessor and method with the same name
//
// If there is an error, it will be of type [loxerr.Errors].
func CheckSemantics(program *ast.Program, opts ...Option) error {
	cfg := newConfig(opts)
	c := &semanticChecker{extraFeatures: cfg.extraFeatures}
	return c.Check(program)
}

type semanticChecker struct {
	extraFeatures bool

	inLoop       bool
	curFunType   funType
	inMethod     bool
	curClassDecl *ast.ClassDecl

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
		c.walkClassDecl(node)
		return false
	case *ast.MethodDecl:
		c.checkNumPropertyAccessorParams(node)
		c.walkFun(node.Function, methodFunType(node))
		c.checkNoStaticInit(node)
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
		c.checkNoInitReturn(node)
	case *ast.FunExpr:
		c.walkFun(node.Function, funTypeFunction)
		return false
	case *ast.IdentExpr:
		c.checkNoBlankAccess(node)
	case *ast.ThisExpr:
		c.checkThisInMethod(node)
	case *ast.SuperExpr:
		c.checkSuperInMethod(node)
		c.checkSuperInSubclass(node)
	case *ast.CallExpr:
		c.checkNumArgs(node.Args)
	case *ast.PropertyExpr:
		c.checkNoBlankPropertyAccess(node.Name)
	case *ast.PropertySetExpr:
		c.checkNoBlankPropertyAccess(node.Name)
		c.checkNoSuperPropertyAssignment(node)
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

	ast.WalkChildren(fun, c.walk)
}

func (c *semanticChecker) walkClassDecl(decl *ast.ClassDecl) {
	prevCurClassDecl := c.curClassDecl
	defer func() { c.curClassDecl = prevCurClassDecl }()
	c.curClassDecl = decl

	c.checkNoSelfReferentialSuperclass(decl)
	c.checkMethods(decl.Methods())

	ast.WalkChildren(decl, c.walk)
}

func (c *semanticChecker) checkMethods(decls []*ast.MethodDecl) {
	fullNames := map[string]bool{}
	methodSeenFirstByIsStatic := map[bool]map[string]bool{false: {}, true: {}}
	accessorSeenFirstByIsStatic := map[bool]map[string]bool{false: {}, true: {}}
	gettersByNameByIsStatic := map[bool]map[string]bool{false: {}, true: {}}
	setterIdentsByNameByIsStatic := map[bool]map[string]*ast.Ident{false: {}, true: {}}
	for _, decl := range decls {
		if !decl.Name.IsValid() {
			continue
		}
		if decl.Name.String() == token.IdentBlank {
			continue
		}
		name := decl.Name.String()

		modifiers := new(strings.Builder)
		for _, modifier := range decl.Modifiers {
			fmt.Fprintf(modifiers, "%s ", modifier.Lexeme)
		}
		fullName := fmt.Sprintf("%s%s", modifiers, name)
		if fullNames[fullName] {
			c.errs.Addf(decl.Name, loxerr.Fatal, "%s%m has already been declared", modifiers, decl.Name)
		}
		fullNames[fullName] = true

		isStatic := decl.HasModifier(token.Static)
		static := ""
		if isStatic {
			static = "static "
		}
		if decl.HasModifier(token.Get, token.Set) {
			switch {
			case decl.HasModifier(token.Get):
				gettersByNameByIsStatic[isStatic][name] = true
			case decl.HasModifier(token.Set):
				setterIdentsByNameByIsStatic[isStatic][name] = decl.Name
			}
			if methodSeenFirstByIsStatic[isStatic][name] {
				c.errs.Addf(decl.Name, loxerr.Fatal, "%s%m has already been declared as a method", static, decl.Name)
			} else {
				accessorSeenFirstByIsStatic[isStatic][name] = true
			}

		} else {
			if accessorSeenFirstByIsStatic[isStatic][name] {
				c.errs.Addf(decl.Name, loxerr.Fatal, "%s%m has already been declared as a property accessor", static, decl.Name)
			} else {
				methodSeenFirstByIsStatic[isStatic][name] = true
			}
		}
	}

	for isStatic, setterIdentsByName := range setterIdentsByNameByIsStatic {
		for name, setterIdent := range setterIdentsByName {
			if !gettersByNameByIsStatic[isStatic][name] {
				c.errs.Addf(setterIdent, loxerr.Fatal, "write-only properties are not allowed")
			}
		}
	}
}

func (c *semanticChecker) checkNumParams(params []*ast.ParamDecl) {
	if len(params) > maxParams {
		c.errs.Addf(params[maxParams], loxerr.Fatal, "cannot define more than %d function parameters", maxParams)
	}
}

func (c *semanticChecker) checkNoStaticInit(decl *ast.MethodDecl) {
	if decl.Name.IsValid() && decl.Name.String() == token.IdentInit && decl.HasModifier(token.Static) {
		c.errs.Addf(decl.Name, loxerr.Fatal, "%s() cannot be static", token.IdentInit)
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

func (c *semanticChecker) checkNoSelfReferentialSuperclass(decl *ast.ClassDecl) {
	if decl.Superclass.IsValid() && decl.Superclass.String() == decl.Name.String() {
		c.errs.Addf(decl.Superclass, loxerr.Fatal, "class cannot inherit from itself")
	}
}

func (c *semanticChecker) checkNumPropertyAccessorParams(decl *ast.MethodDecl) {
	params := decl.GetParams()
	switch {
	case decl.HasModifier(token.Get) && len(params) > 0:
		c.errs.AddSpanningRangesf(params[0], params[len(params)-1], loxerr.Fatal, "property getter cannot have parameters")
	case decl.HasModifier(token.Set):
		if len(params) == 0 && decl.Name.IsValid() {
			c.errs.Addf(decl.Name, loxerr.Fatal, "property setter must have a parameter")
		} else if len(params) > 1 {
			c.errs.AddSpanningRangesf(params[1], params[len(params)-1], loxerr.Fatal, "property setter can only have one parameter")
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

func (c *semanticChecker) checkNoInitReturn(stmt *ast.ReturnStmt) {
	if stmt.Value != nil && c.curFunType.IsInit() {
		c.errs.Addf(stmt, loxerr.Fatal, "%s() cannot return a value", token.IdentInit)
	}
}

func (c *semanticChecker) checkNoBlankAccess(expr *ast.IdentExpr) {
	if c.extraFeatures && expr.Ident.IsValid() && expr.Ident.String() == token.IdentBlank {
		c.errs.Addf(expr.Ident, loxerr.Fatal, "'%s' cannot be used as a value", token.IdentBlank)
	}
}

func (c *semanticChecker) checkNoBlankPropertyAccess(ident *ast.Ident) {
	if c.extraFeatures && ident.IsValid() && ident.String() == token.IdentBlank {
		c.errs.Addf(ident, loxerr.Fatal, "'%s' is not a valid property name", token.IdentBlank)
	}
}

func (c *semanticChecker) checkNoSuperPropertyAssignment(expr *ast.PropertySetExpr) {
	if _, ok := expr.Object.(*ast.SuperExpr); ok {
		c.errs.Addf(expr.Name, loxerr.Fatal, "property assignment is not valid for %m object", token.Super)
	}
}

func (c *semanticChecker) checkThisInMethod(expr *ast.ThisExpr) {
	if !c.inMethod {
		c.errs.Addf(expr, loxerr.Fatal, "%m can only be used inside a method definition", token.This)
	}
}

func (c *semanticChecker) checkSuperInMethod(expr *ast.SuperExpr) {
	if !c.inMethod {
		c.errs.Addf(expr.Super, loxerr.Fatal, "%m can only be used inside a method definition", token.Super)
	}
}

func (c *semanticChecker) checkSuperInSubclass(expr *ast.SuperExpr) {
	if c.curClassDecl != nil && !c.curClassDecl.Superclass.IsValid() {
		c.errs.Addf(expr.Super, loxerr.Fatal, "%m can only be used inside a subclass", token.Super)
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
	funTypeInitFlag
)

func (f funType) IsMethod() bool {
	return f&funTypeMethodFlag != 0
}

func (f funType) IsInit() bool {
	return f&funTypeInitFlag != 0
}

func methodFunType(decl *ast.MethodDecl) funType {
	typ := funTypeFunction | funTypeMethodFlag
	if decl.IsInit() {
		typ |= funTypeInitFlag
	}
	return typ
}
