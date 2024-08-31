package main

import (
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
)

const indentSize = 2

func format(node ast.Node) string {
	switch node := node.(type) {
	case ast.Program:
		return formatProgram(node)
	case ast.VarDecl:
		return formatVarDecl(node)
	case ast.FunDecl:
		return formatFunDecl(node)
	case ast.ClassDecl:
		return formatClassDecl(node)
	case ast.MethodDecl:
		return formatMethodDecl(node)
	case ast.ExprStmt:
		return formatExprStmt(node)
	case ast.PrintStmt:
		return formatPrintStmt(node)
	case ast.BlockStmt:
		return formatBlockStmt(node)
	case ast.IfStmt:
		return formatIfStmt(node)
	case ast.WhileStmt:
		return formatWhileStmt(node)
	case ast.ForStmt:
		return formatForStmt(node)
	case ast.BreakStmt:
		return formatBreakStmt(node)
	case ast.ContinueStmt:
		return formatContinueStmt(node)
	case ast.ReturnStmt:
		return formatReturnStmt(node)
	case ast.FunExpr:
		return formatFunExpr(node)
	case ast.GroupExpr:
		return formatGroupExpr(node)
	case ast.LiteralExpr:
		return formatLiteralExpr(node)
	case ast.VariableExpr:
		return formatVariableExpr(node)
	case ast.ThisExpr:
		return formatThisExpr(node)
	case ast.CallExpr:
		return formatCallExpr(node)
	case ast.GetExpr:
		return formatGetExpr(node)
	case ast.UnaryExpr:
		return formatUnaryExpr(node)
	case ast.BinaryExpr:
		return formatBinaryExpr(node)
	case ast.TernaryExpr:
		return formatTernaryExpr(node)
	case ast.AssignmentExpr:
		return formatAssignmentExpr(node)
	case ast.SetExpr:
		return formatSetExpr(node)
	default:
		panic(fmt.Sprintf("unexpected node type: %T", node))
	}
}

func formatProgram(program ast.Program) string {
	return fmt.Sprint(formatStmts(program.Stmts), "\n")
}

func formatStmts[T ast.Stmt](stmts []T) string {
	var b strings.Builder
	for i, stmt := range stmts {
		fmt.Fprint(&b, format(stmt))
		if i < len(stmts)-1 {
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func formatVarDecl(decl ast.VarDecl) string {
	var b strings.Builder
	fmt.Fprintf(&b, "var %s", decl.Name.Lexeme)
	if decl.Initialiser != nil {
		fmt.Fprintf(&b, " = %s", format(decl.Initialiser))
	}
	fmt.Fprint(&b, ";")
	return b.String()
}

func formatFunDecl(decl ast.FunDecl) string {
	return fmt.Sprintf("fun %s%s", decl.Name.Lexeme, formatFun(decl.Params, decl.Body))
}

func formatFun(params []token.Token, body []ast.Stmt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "(")
	for i, param := range params {
		fmt.Fprintf(&b, "%s", param.Lexeme)
		if i < len(params)-1 {
			fmt.Fprint(&b, ", ")
		}
	}
	fmt.Fprintf(&b, ") {\n%s\n}", indent(formatStmts(body)))
	return b.String()
}

func formatClassDecl(decl ast.ClassDecl) string {
	return fmt.Sprintf("class %s {\n%s\n}", decl.Name.Lexeme, indent(formatStmts(decl.Methods)))
}

func formatMethodDecl(decl ast.MethodDecl) string {
	var b strings.Builder
	for _, modifier := range decl.Modifiers {
		fmt.Fprintf(&b, "%s ", modifier.Lexeme)
	}
	fmt.Fprintf(&b, "%s%s", decl.Name.Lexeme, formatFun(decl.Params, decl.Body))
	return b.String()
}

func formatExprStmt(stmt ast.ExprStmt) string {
	return fmt.Sprintf("%s;", format(stmt.Expr))
}

func formatPrintStmt(stmt ast.PrintStmt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "print %s;", format(stmt.Expr))
	return b.String()
}

func formatBlockStmt(stmt ast.BlockStmt) string {
	var b strings.Builder
	fmt.Fprint(&b, "{\n", indent(formatStmts(stmt.Stmts)), "\n}")
	return b.String()
}

func formatIfStmt(stmt ast.IfStmt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "if (%s)", format(stmt.Condition))
	var thenIsBlock bool
	if _, thenIsBlock = stmt.Then.(ast.BlockStmt); thenIsBlock {
		fmt.Fprint(&b, " ", format(stmt.Then))
	} else {
		fmt.Fprint(&b, "\n", indent(format(stmt.Then)))
	}
	if stmt.Else != nil {
		if thenIsBlock {
			fmt.Fprint(&b, " ")
		} else {
			fmt.Fprint(&b, "\n")
		}
		switch stmt.Else.(type) {
		case ast.IfStmt, ast.BlockStmt:
			fmt.Fprint(&b, "else ", format(stmt.Else))
		default:
			fmt.Fprint(&b, "else\n", indent(format(stmt.Else)))
		}
	}
	return b.String()
}

func formatWhileStmt(stmt ast.WhileStmt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "while (%s)", format(stmt.Condition))
	if _, ok := stmt.Body.(ast.BlockStmt); ok {
		fmt.Fprintf(&b, " %s", format(stmt.Body))
	} else {
		fmt.Fprintf(&b, "\n%s", indent(format(stmt.Body)))
	}
	return b.String()
}

func formatForStmt(stmt ast.ForStmt) string {
	var b strings.Builder
	fmt.Fprint(&b, "for (")
	if stmt.Initialise != nil {
		fmt.Fprintf(&b, "%s", format(stmt.Initialise))
	} else {
		fmt.Fprint(&b, ";")
	}
	if stmt.Condition != nil {
		fmt.Fprintf(&b, " %s", format(stmt.Condition))
	}
	fmt.Fprint(&b, ";")
	if stmt.Update != nil {
		fmt.Fprintf(&b, " %s", format(stmt.Update))
	}
	fmt.Fprint(&b, ")")
	if _, ok := stmt.Body.(ast.BlockStmt); ok {
		fmt.Fprintf(&b, " %s", format(stmt.Body))
	} else {
		fmt.Fprintf(&b, "\n%s", indent(format(stmt.Body)))
	}
	return b.String()
}

func formatBreakStmt(ast.BreakStmt) string {
	return "break;"
}

func formatContinueStmt(ast.ContinueStmt) string {
	return "continue;"
}

func formatReturnStmt(stmt ast.ReturnStmt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "return")
	if stmt.Value != nil {
		fmt.Fprintf(&b, " %s", format(stmt.Value))
	}
	fmt.Fprintf(&b, ";")
	return b.String()
}

func formatFunExpr(expr ast.FunExpr) string {
	return fmt.Sprintf("fun%s", formatFun(expr.Params, expr.Body))
}

func formatGroupExpr(expr ast.GroupExpr) string {
	return fmt.Sprintf("(%s)", format(expr.Expr))
}

func formatLiteralExpr(expr ast.LiteralExpr) string {
	return expr.Value.Lexeme
}

func formatVariableExpr(expr ast.VariableExpr) string {
	return expr.Name.Lexeme
}

func formatThisExpr(ast.ThisExpr) string {
	return "this"
}

func formatCallExpr(expr ast.CallExpr) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s(", format(expr.Callee))
	for i, arg := range expr.Args {
		fmt.Fprint(&b, format(arg))
		if i < len(expr.Args)-1 {
			fmt.Fprint(&b, ", ")
		}
	}
	fmt.Fprint(&b, ")")
	return b.String()
}

func formatGetExpr(expr ast.GetExpr) string {
	return fmt.Sprintf("%s.%s", format(expr.Object), expr.Name.Lexeme)
}

func formatUnaryExpr(expr ast.UnaryExpr) string {
	return fmt.Sprintf("%s%s", expr.Op.Lexeme, format(expr.Right))
}

func formatBinaryExpr(expr ast.BinaryExpr) string {
	return fmt.Sprintf("%s %s %s", format(expr.Left), expr.Op.Lexeme, format(expr.Right))
}

func formatTernaryExpr(expr ast.TernaryExpr) string {
	return fmt.Sprint(format(expr.Condition), " ? ", format(expr.Then), " : ", format(expr.Else))
}

func formatAssignmentExpr(expr ast.AssignmentExpr) string {
	return fmt.Sprintf("%s = %s", expr.Left.Lexeme, format(expr.Right))
}

func formatSetExpr(expr ast.SetExpr) string {
	return fmt.Sprintf("%s.%s = %s", format(expr.Object), expr.Name.Lexeme, format(expr.Value))
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat(" ", indentSize) + line
	}
	return strings.Join(lines, "\n")
}
