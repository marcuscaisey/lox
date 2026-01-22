// Package format implements canonical formatting of Lox code.
package format

import (
	"fmt"
	"strings"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/token"
)

const indentSize = 2

// Node formats node in canonical Lox style and returns the result. node is expected to be a syntactically correct.
func Node(node ast.Node) string {
	switch node := node.(type) {
	case *ast.Program:
		return formatProgram(node)
	case *ast.Ident:
		return formatIdent(node)
	case *ast.IllegalStmt:
		panic("IllegalStmt cannot be formatted")
	case *ast.Comment:
		return formatComment(node)
	case *ast.CommentedStmt:
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
	case *ast.SuperExpr:
		return formatSuperExpr(node)
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
	}
	panic("unreachable")
}

func formatIdent(ident *ast.Ident) string {
	return ident.String()
}

func formatProgram(program *ast.Program) string {
	return fmt.Sprint(formatStmts(program.Stmts), "\n")
}

func formatStmts[T ast.Stmt](stmts []T) string {
	b := new(strings.Builder)
	for i, stmt := range stmts {
		fmt.Fprint(b, Node(stmt))
		if i < len(stmts)-1 {
			fmt.Fprintln(b)
			if stmts[i+1].Start().Line-stmts[i].End().Line > 1 {
				fmt.Fprintln(b)
			}
		}
	}
	return b.String()
}

func formatComment(stmt *ast.Comment) string {
	return stmt.Comment.Lexeme
}

func formatCommentedStmt(stmt *ast.CommentedStmt) string {
	return fmt.Sprint(Node(stmt.Stmt), " ", stmt.Comment.Comment.Lexeme)
}

func formatVarDecl(decl *ast.VarDecl) string {
	if decl.Initialiser != nil {
		return fmt.Sprint(token.Var, " ", Node(decl.Name), " ", token.Equal, " ", Node(decl.Initialiser), token.Semicolon)
	} else {
		return fmt.Sprint(token.Var, " ", Node(decl.Name), token.Semicolon)
	}
}

func formatFunDecl(decl *ast.FunDecl) string {
	b := new(strings.Builder)
	if len(decl.DocComments) > 0 {
		fmt.Fprintln(b, formatStmts(decl.DocComments))
	}
	fmt.Fprint(b, token.Fun, " ", Node(decl.Name), Node(decl.Function))
	return b.String()
}

func formatFun(fun *ast.Function) string {
	b := new(strings.Builder)
	fmt.Fprint(b, token.LeftParen)
	for i, param := range fun.Params {
		fmt.Fprint(b, Node(param))
		if i < len(fun.Params)-1 {
			fmt.Fprint(b, token.Comma, " ")
		}
	}
	fmt.Fprint(b, token.RightParen, " ", formatBlock(fun.Body.Stmts))
	return b.String()
}

func formatParamDecl(decl *ast.ParamDecl) string {
	return formatIdent(decl.Name)
}

func formatClassDecl(decl *ast.ClassDecl) string {
	b := new(strings.Builder)
	if len(decl.DocComments) > 0 {
		fmt.Fprintln(b, formatStmts(decl.DocComments))
	}
	fmt.Fprint(b, token.Class, " ", Node(decl.Name), " ")
	if decl.Superclass.IsValid() {
		fmt.Fprint(b, token.Less, " ", Node(decl.Superclass), " ")
	}
	fmt.Fprint(b, Node(decl.Body))
	return b.String()
}

func formatMethodDecl(decl *ast.MethodDecl) string {
	b := new(strings.Builder)
	if len(decl.DocComments) > 0 {
		fmt.Fprintln(b, formatStmts(decl.DocComments))
	}
	for _, modifier := range decl.Modifiers {
		fmt.Fprint(b, modifier.Type, " ")
	}
	fmt.Fprint(b, Node(decl.Name), Node(decl.Function))
	return b.String()
}

func formatExprStmt(stmt *ast.ExprStmt) string {
	return fmt.Sprint(Node(stmt.Expr), token.Semicolon)
}

func formatPrintStmt(stmt *ast.PrintStmt) string {
	return fmt.Sprint(token.Print, " ", Node(stmt.Expr), token.Semicolon)
}

func formatBlockStmt(stmt *ast.Block) string {
	return formatBlock(stmt.Stmts)
}

func formatBlock[T ast.Stmt](stmts []T) string {
	if len(stmts) > 0 {
		return fmt.Sprint(token.LeftBrace, "\n", indent(formatStmts(stmts)), "\n", token.RightBrace)
	} else {
		return fmt.Sprint(token.LeftBrace, "", token.RightBrace)
	}
}

func formatIfStmt(stmt *ast.IfStmt) string {
	b := new(strings.Builder)
	fmt.Fprint(b, token.If, " ", token.LeftParen, Node(stmt.Condition), token.RightParen)
	var thenIsBlock bool
	if _, thenIsBlock = stmt.Then.(*ast.Block); thenIsBlock {
		fmt.Fprint(b, " ", Node(stmt.Then))
	} else {
		fmt.Fprint(b, "\n", indent(Node(stmt.Then)))
	}
	if stmt.Else != nil {
		if thenIsBlock {
			fmt.Fprint(b, " ")
		} else {
			fmt.Fprint(b, "\n")
		}
		switch stmt.Else.(type) {
		case *ast.IfStmt, *ast.Block:
			fmt.Fprint(b, token.Else, " ", Node(stmt.Else))
		default:
			fmt.Fprint(b, token.Else, "\n", indent(Node(stmt.Else)))
		}
	}
	return b.String()
}

func formatWhileStmt(stmt *ast.WhileStmt) string {
	if _, ok := stmt.Body.(*ast.Block); ok {
		return fmt.Sprint(token.While, " ", token.LeftParen, Node(stmt.Condition), token.RightParen, " ", Node(stmt.Body))
	} else {
		return fmt.Sprint(token.While, " ", token.LeftParen, Node(stmt.Condition), token.RightParen, "\n", indent(Node(stmt.Body)))
	}
}

func formatForStmt(stmt *ast.ForStmt) string {
	b := new(strings.Builder)
	fmt.Fprint(b, token.For, " ", token.LeftParen)
	if stmt.Initialise != nil {
		fmt.Fprint(b, Node(stmt.Initialise))
	} else {
		fmt.Fprint(b, token.Semicolon)
	}
	if stmt.Condition != nil {
		fmt.Fprint(b, " ", Node(stmt.Condition))
	}
	fmt.Fprint(b, token.Semicolon)
	if stmt.Update != nil {
		fmt.Fprint(b, " ", Node(stmt.Update))
	}
	fmt.Fprint(b, token.RightParen)
	if _, ok := stmt.Body.(*ast.Block); ok {
		fmt.Fprint(b, " ", Node(stmt.Body))
	} else {
		fmt.Fprint(b, "\n", indent(Node(stmt.Body)))
	}
	return b.String()
}

func formatBreakStmt(*ast.BreakStmt) string {
	return fmt.Sprint(token.Break, "", token.Semicolon)
}

func formatContinueStmt(*ast.ContinueStmt) string {
	return fmt.Sprint(token.Continue, "", token.Semicolon)
}

func formatReturnStmt(stmt *ast.ReturnStmt) string {
	if stmt.Value != nil {
		return fmt.Sprint(token.Return, " ", Node(stmt.Value), token.Semicolon)
	} else {
		return fmt.Sprint(token.Return, "", token.Semicolon)
	}
}

func formatFunExpr(expr *ast.FunExpr) string {
	return fmt.Sprint(token.Fun, Node(expr.Function))
}

func formatGroupExpr(expr *ast.GroupExpr) string {
	return fmt.Sprint(token.LeftParen, Node(expr.Expr), token.RightParen)
}

func formatLiteralExpr(expr *ast.LiteralExpr) string {
	return expr.Value.Lexeme
}

func formatIdentExpr(expr *ast.IdentExpr) string {
	return expr.Ident.String()
}

func formatThisExpr(*ast.ThisExpr) string {
	return token.This.String()
}

func formatSuperExpr(*ast.SuperExpr) string {
	return token.Super.String()
}

func formatCallExpr(expr *ast.CallExpr) string {
	b := new(strings.Builder)
	fmt.Fprint(b, Node(expr.Callee), token.LeftParen)
	for i, arg := range expr.Args {
		fmt.Fprint(b, Node(arg))
		if i < len(expr.Args)-1 {
			fmt.Fprint(b, token.Comma, " ")
		}
	}
	fmt.Fprint(b, token.RightParen)
	return b.String()
}

func formatGetExpr(expr *ast.GetExpr) string {
	return fmt.Sprint(Node(expr.Object), token.Dot, Node(expr.Name))
}

func formatUnaryExpr(expr *ast.UnaryExpr) string {
	return fmt.Sprint(expr.Op.Lexeme, Node(expr.Right))
}

func formatBinaryExpr(expr *ast.BinaryExpr) string {
	leftSpace := " "
	if expr.Op.Type == token.Comma {
		// Comma operator is a special case where we don't want a space before it. A binary expression with a comma
		// operator should be formatted as "a, b" rather than "a , b".
		leftSpace = ""
	}
	return fmt.Sprint(Node(expr.Left), leftSpace, expr.Op.Lexeme, " ", Node(expr.Right))
}

func formatTernaryExpr(expr *ast.TernaryExpr) string {
	return fmt.Sprint(Node(expr.Condition), " ", token.Question, " ", Node(expr.Then), " ", token.Colon, " ", Node(expr.Else))
}

func formatAssignmentExpr(expr *ast.AssignmentExpr) string {
	return fmt.Sprint(Node(expr.Left), " ", token.Equal, " ", Node(expr.Right))
}

func formatSetExpr(expr *ast.SetExpr) string {
	return fmt.Sprint(Node(expr.Object), token.Dot, Node(expr.Name), " ", token.Equal, " ", Node(expr.Value))
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
