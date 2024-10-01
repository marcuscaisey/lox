package main

import (
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/golox/ast"
)

const indentSize = 2

func format(node ast.Node) string {
	switch node := node.(type) {
	case ast.Program:
		return formatProgram(node)
	case ast.CommentStmt:
		return formatCommentStmt(node)
	case ast.InlineCommentStmt:
		return formatCommentedStmt(node)
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
			if stmts[i+1].Start().Line-stmts[i].End().Line > 1 {
				fmt.Fprintln(&b)
			}
		}
	}
	return b.String()
}

func formatCommentStmt(stmt ast.CommentStmt) string {
	return stmt.Comment.Lexeme
}

func formatCommentedStmt(stmt ast.InlineCommentStmt) string {
	return fmt.Sprintf("%s %s", format(stmt.Stmt), stmt.Comment.Lexeme)
}

func formatVarDecl(decl ast.VarDecl) string {
	if decl.Initialiser != nil {
		return fmt.Sprintf("var %s = %s;", decl.Name.Lexeme, format(decl.Initialiser))
	} else {
		return fmt.Sprintf("var %s;", decl.Name.Lexeme)
	}
}

func formatFunDecl(decl ast.FunDecl) string {
	return fmt.Sprintf("fun %s%s", decl.Name.Lexeme, formatFun(decl.Function))
}

func formatFun(fun ast.Function) string {
	var b strings.Builder
	fmt.Fprintf(&b, "(")
	for i, param := range fun.Params {
		fmt.Fprintf(&b, "%s", param.Lexeme)
		if i < len(fun.Params)-1 {
			fmt.Fprint(&b, ", ")
		}
	}
	fmt.Fprintf(&b, ") %s", formatBlock(fun.Body.Stmts))
	return b.String()
}

func formatClassDecl(decl ast.ClassDecl) string {
	return fmt.Sprintf("class %s %s", decl.Name.Lexeme, formatBlock(decl.Body))
}

func formatMethodDecl(decl ast.MethodDecl) string {
	var b strings.Builder
	for _, modifier := range decl.Modifiers {
		fmt.Fprintf(&b, "%s ", modifier.Lexeme)
	}
	fmt.Fprintf(&b, "%s%s", decl.Name.Lexeme, formatFun(decl.Function))
	return b.String()
}

func formatExprStmt(stmt ast.ExprStmt) string {
	return fmt.Sprintf("%s;", format(stmt.Expr))
}

func formatPrintStmt(stmt ast.PrintStmt) string {
	return fmt.Sprintf("print %s;", format(stmt.Expr))
}

func formatBlockStmt(stmt ast.BlockStmt) string {
	return formatBlock(stmt.Stmts)
}

func formatBlock[T ast.Stmt](stmts []T) string {
	if len(stmts) > 0 {
		return fmt.Sprintf("{\n%s\n}", indent(formatStmts(stmts)))
	} else {
		return "{}"
	}
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
	if _, ok := stmt.Body.(ast.BlockStmt); ok {
		return fmt.Sprintf("while (%s) %s", format(stmt.Condition), format(stmt.Body))
	} else {
		return fmt.Sprintf("while (%s)\n%s", format(stmt.Condition), indent(format(stmt.Body)))
	}
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
	if stmt.Value != nil {
		return fmt.Sprintf("return %s;", format(stmt.Value))
	} else {
		return "return;"
	}
}

func formatFunExpr(expr ast.FunExpr) string {
	return fmt.Sprintf("fun%s", formatFun(expr.Function))
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
		if line != "" {
			lines[i] = strings.Repeat(" ", indentSize) + line
		}
	}
	return strings.Join(lines, "\n")
}
