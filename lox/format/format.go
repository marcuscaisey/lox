// Package format implements canonical formatting of Lox code.
package format

import (
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/lox/ast"
	"github.com/marcuscaisey/lox/lox/token"
)

const indentSize = 4

// Node formats node in canonical Lox style and returns the result. node is expected to be a syntactically correct.
func Node(node ast.Node) string {
	switch node := node.(type) {
	case *ast.Program:
		return formatProgram(node)
	case *ast.Ident:
		return formatIdent(node)
	case *ast.Comment:
		return formatComment(node)
	case *ast.InlineComment:
		return formatCommentedStmt(node)
	case *ast.VarDecl:
		return formatVarDecl(node)
	case *ast.FunDecl:
		return formatFunDecl(node)
	case *ast.Function:
		return formatFun(node)
	case *ast.ParamDecl:
		return formatParamDecl(node)
	case *ast.ClassDecl:
		return formatClassDecl(node)
	case *ast.MethodDecl:
		return formatMethodDecl(node)
	case *ast.ExprStmt:
		return formatExprStmt(node)
	case *ast.PrintStmt:
		return formatPrintStmt(node)
	case *ast.Block:
		return formatBlockStmt(node)
	case *ast.IfStmt:
		return formatIfStmt(node)
	case *ast.WhileStmt:
		return formatWhileStmt(node)
	case *ast.ForStmt:
		return formatForStmt(node)
	case *ast.BreakStmt:
		return formatBreakStmt(node)
	case *ast.ContinueStmt:
		return formatContinueStmt(node)
	case *ast.ReturnStmt:
		return formatReturnStmt(node)
	case *ast.FunExpr:
		return formatFunExpr(node)
	case *ast.GroupExpr:
		return formatGroupExpr(node)
	case *ast.LiteralExpr:
		return formatLiteralExpr(node)
	case *ast.IdentExpr:
		return formatIdentExpr(node)
	case *ast.ThisExpr:
		return formatThisExpr(node)
	case *ast.CallExpr:
		return formatCallExpr(node)
	case *ast.GetExpr:
		return formatGetExpr(node)
	case *ast.UnaryExpr:
		return formatUnaryExpr(node)
	case *ast.BinaryExpr:
		return formatBinaryExpr(node)
	case *ast.TernaryExpr:
		return formatTernaryExpr(node)
	case *ast.AssignmentExpr:
		return formatAssignmentExpr(node)
	case *ast.SetExpr:
		return formatSetExpr(node)
	case *ast.IllegalStmt:
		panic("IllegalStmt cannot be formatted")
	}
	panic("unreachable")
}

func formatIdent(ident *ast.Ident) string {
	return ident.Token.Lexeme
}

func formatProgram(program *ast.Program) string {
	return fmt.Sprint(formatStmts(program.Stmts), "\n")
}

func formatStmts[T ast.Stmt](stmts []T) string {
	var b strings.Builder
	for i, stmt := range stmts {
		fmt.Fprint(&b, Node(stmt))
		if i < len(stmts)-1 {
			fmt.Fprintln(&b)
			if stmts[i+1].Start().Line-stmts[i].End().Line > 1 {
				fmt.Fprintln(&b)
			}
		}
	}
	return b.String()
}

func formatComment(stmt *ast.Comment) string {
	return stmt.Comment.Lexeme
}

func formatCommentedStmt(stmt *ast.InlineComment) string {
	return fmt.Sprintf("%s %s", Node(stmt.Stmt), stmt.Comment.Lexeme)
}

func formatVarDecl(decl *ast.VarDecl) string {
	if decl.Initialiser != nil {
		return fmt.Sprintf("var %s = %s;", Node(decl.Name), Node(decl.Initialiser))
	} else {
		return fmt.Sprintf("var %s;", Node(decl.Name))
	}
}

func formatFunDecl(decl *ast.FunDecl) string {
	return fmt.Sprintf("fun %s%s", Node(decl.Name), Node(decl.Function))
}

func formatFun(fun *ast.Function) string {
	var b strings.Builder
	fmt.Fprintf(&b, "(")
	for i, param := range fun.Params {
		fmt.Fprint(&b, Node(param))
		if i < len(fun.Params)-1 {
			fmt.Fprint(&b, ", ")
		}
	}
	fmt.Fprintf(&b, ") %s", formatBlock(fun.Body.Stmts))
	return b.String()
}

func formatParamDecl(decl *ast.ParamDecl) string {
	return formatIdent(decl.Name)
}

func formatClassDecl(decl *ast.ClassDecl) string {
	return fmt.Sprintf("class %s %s", Node(decl.Name), formatBlock(decl.Body))
}

func formatMethodDecl(decl *ast.MethodDecl) string {
	var b strings.Builder
	for _, modifier := range decl.Modifiers {
		fmt.Fprintf(&b, "%s ", modifier.Lexeme)
	}
	fmt.Fprintf(&b, "%s%s", Node(decl.Name), Node(decl.Function))
	return b.String()
}

func formatExprStmt(stmt *ast.ExprStmt) string {
	return fmt.Sprintf("%s;", Node(stmt.Expr))
}

func formatPrintStmt(stmt *ast.PrintStmt) string {
	return fmt.Sprintf("print %s;", Node(stmt.Expr))
}

func formatBlockStmt(stmt *ast.Block) string {
	return formatBlock(stmt.Stmts)
}

func formatBlock[T ast.Stmt](stmts []T) string {
	if len(stmts) > 0 {
		return fmt.Sprintf("{\n%s\n}", indent(formatStmts(stmts)))
	} else {
		return "{}"
	}
}

func formatIfStmt(stmt *ast.IfStmt) string {
	var b strings.Builder
	fmt.Fprintf(&b, "if (%s)", Node(stmt.Condition))
	var thenIsBlock bool
	if _, thenIsBlock = stmt.Then.(*ast.Block); thenIsBlock {
		fmt.Fprint(&b, " ", Node(stmt.Then))
	} else {
		fmt.Fprint(&b, "\n", indent(Node(stmt.Then)))
	}
	if stmt.Else != nil {
		if thenIsBlock {
			fmt.Fprint(&b, " ")
		} else {
			fmt.Fprint(&b, "\n")
		}
		switch stmt.Else.(type) {
		case *ast.IfStmt, *ast.Block:
			fmt.Fprint(&b, "else ", Node(stmt.Else))
		default:
			fmt.Fprint(&b, "else\n", indent(Node(stmt.Else)))
		}
	}
	return b.String()
}

func formatWhileStmt(stmt *ast.WhileStmt) string {
	if _, ok := stmt.Body.(*ast.Block); ok {
		return fmt.Sprintf("while (%s) %s", Node(stmt.Condition), Node(stmt.Body))
	} else {
		return fmt.Sprintf("while (%s)\n%s", Node(stmt.Condition), indent(Node(stmt.Body)))
	}
}

func formatForStmt(stmt *ast.ForStmt) string {
	var b strings.Builder
	fmt.Fprint(&b, "for (")
	if stmt.Initialise != nil {
		fmt.Fprintf(&b, "%s", Node(stmt.Initialise))
	} else {
		fmt.Fprint(&b, ";")
	}
	if stmt.Condition != nil {
		fmt.Fprintf(&b, " %s", Node(stmt.Condition))
	}
	fmt.Fprint(&b, ";")
	if stmt.Update != nil {
		fmt.Fprintf(&b, " %s", Node(stmt.Update))
	}
	fmt.Fprint(&b, ")")
	if _, ok := stmt.Body.(*ast.Block); ok {
		fmt.Fprintf(&b, " %s", Node(stmt.Body))
	} else {
		fmt.Fprintf(&b, "\n%s", indent(Node(stmt.Body)))
	}
	return b.String()
}

func formatBreakStmt(*ast.BreakStmt) string {
	return "break;"
}

func formatContinueStmt(*ast.ContinueStmt) string {
	return "continue;"
}

func formatReturnStmt(stmt *ast.ReturnStmt) string {
	if stmt.Value != nil {
		return fmt.Sprintf("return %s;", Node(stmt.Value))
	} else {
		return "return;"
	}
}

func formatFunExpr(expr *ast.FunExpr) string {
	return fmt.Sprintf("fun%s", Node(expr.Function))
}

func formatGroupExpr(expr *ast.GroupExpr) string {
	return fmt.Sprintf("(%s)", Node(expr.Expr))
}

func formatLiteralExpr(expr *ast.LiteralExpr) string {
	return expr.Value.Lexeme
}

func formatIdentExpr(expr *ast.IdentExpr) string {
	return expr.Ident.Token.Lexeme
}

func formatThisExpr(*ast.ThisExpr) string {
	return "this"
}

func formatCallExpr(expr *ast.CallExpr) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s(", Node(expr.Callee))
	for i, arg := range expr.Args {
		fmt.Fprint(&b, Node(arg))
		if i < len(expr.Args)-1 {
			fmt.Fprint(&b, ", ")
		}
	}
	fmt.Fprint(&b, ")")
	return b.String()
}

func formatGetExpr(expr *ast.GetExpr) string {
	return fmt.Sprintf("%s.%s", Node(expr.Object), Node(expr.Name))
}

func formatUnaryExpr(expr *ast.UnaryExpr) string {
	return fmt.Sprintf("%s%s", expr.Op.Lexeme, Node(expr.Right))
}

func formatBinaryExpr(expr *ast.BinaryExpr) string {
	leftSpace := " "
	if expr.Op.Type == token.Comma {
		// Comma operator is a special case where we don't want a space before it. A binary expression with a comma
		// operator should be formatted as "a, b" rather than "a , b".
		leftSpace = ""
	}
	return fmt.Sprintf("%s%s%s %s", Node(expr.Left), leftSpace, expr.Op.Lexeme, Node(expr.Right))
}

func formatTernaryExpr(expr *ast.TernaryExpr) string {
	return fmt.Sprint(Node(expr.Condition), " ? ", Node(expr.Then), " : ", Node(expr.Else))
}

func formatAssignmentExpr(expr *ast.AssignmentExpr) string {
	return fmt.Sprintf("%s = %s", Node(expr.Left), Node(expr.Right))
}

func formatSetExpr(expr *ast.SetExpr) string {
	return fmt.Sprintf("%s.%s = %s", Node(expr.Object), Node(expr.Name), Node(expr.Value))
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
