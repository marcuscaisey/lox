package ast

// Walk traverses an AST in depth-first order: It starts by calling f(node); node must not be nil. If f returns true,
// Walk invokes f recursively for each of the non-nil children of node.
func Walk(node Node, f func(Node) bool) {
	if isNil(node) || !f(node) {
		return
	}
	switch node := node.(type) {
	case *Program:
		walkSlice(node.Stmts, f)
	case *Ident:
	case *Comment:
	case *CommentedStmt:
		Walk(node.Stmt, f)
	case *VarDecl:
		Walk(node.Name, f)
		Walk(node.Initialiser, f)
	case *FunDecl:
		Walk(node.Name, f)
		Walk(node.Function, f)
	case *Function:
		walkSlice(node.Params, f)
		Walk(Node(node.Body), f)
	case *ParamDecl:
		Walk(node.Name, f)
	case *ClassDecl:
		Walk(node.Name, f)
		Walk(node.Body, f)
	case *MethodDecl:
		Walk(node.Name, f)
		Walk(node.Function, f)
	case *ExprStmt:
		Walk(node.Expr, f)
	case *PrintStmt:
		Walk(node.Expr, f)
	case *Block:
		walkSlice(node.Stmts, f)
	case *IfStmt:
		Walk(node.Condition, f)
		Walk(node.Then, f)
		Walk(node.Else, f)
	case *WhileStmt:
		Walk(node.Condition, f)
		Walk(node.Body, f)
	case *ForStmt:
		Walk(node.Initialise, f)
		Walk(node.Condition, f)
		Walk(node.Update, f)
		Walk(node.Body, f)
	case *BreakStmt:
	case *ContinueStmt:
	case *ReturnStmt:
		Walk(node.Value, f)
	case *FunExpr:
		Walk(node.Function, f)
	case *GroupExpr:
		Walk(node.Expr, f)
	case *LiteralExpr:
	case *IdentExpr:
		Walk(node.Ident, f)
	case *ThisExpr:
	case *CallExpr:
		Walk(node.Callee, f)
		walkSlice(node.Args, f)
	case *GetExpr:
		Walk(node.Object, f)
		Walk(node.Name, f)
	case *UnaryExpr:
		Walk(node.Right, f)
	case *BinaryExpr:
		Walk(node.Left, f)
		Walk(node.Right, f)
	case *TernaryExpr:
		Walk(node.Condition, f)
		Walk(node.Then, f)
		Walk(node.Else, f)
	case *AssignmentExpr:
		Walk(node.Left, f)
		Walk(node.Right, f)
	case *SetExpr:
		Walk(node.Object, f)
		Walk(node.Name, f)
		Walk(node.Value, f)
	}
}

func isNil(node Node) bool {
	switch node := node.(type) {
	case *Program:
		return node == nil
	case *Ident:
		return node == nil
	case *Comment:
		return node == nil
	case *CommentedStmt:
		return node == nil
	case *VarDecl:
		return node == nil
	case *FunDecl:
		return node == nil
	case *Function:
		return node == nil
	case *ParamDecl:
		return node == nil
	case *ClassDecl:
		return node == nil
	case *MethodDecl:
		return node == nil
	case *ExprStmt:
		return node == nil
	case *PrintStmt:
		return node == nil
	case *Block:
		return node == nil
	case *IfStmt:
		return node == nil
	case *WhileStmt:
		return node == nil
	case *ForStmt:
		return node == nil
	case *BreakStmt:
		return node == nil
	case *ContinueStmt:
		return node == nil
	case *ReturnStmt:
		return node == nil
	case *FunExpr:
		return node == nil
	case *GroupExpr:
		return node == nil
	case *LiteralExpr:
		return node == nil
	case *IdentExpr:
		return node == nil
	case *ThisExpr:
		return node == nil
	case *CallExpr:
		return node == nil
	case *GetExpr:
		return node == nil
	case *UnaryExpr:
		return node == nil
	case *BinaryExpr:
		return node == nil
	case *TernaryExpr:
		return node == nil
	case *AssignmentExpr:
		return node == nil
	case *SetExpr:
		return node == nil
	case nil:
		return true
	}
	return false
}

func walkSlice[T Node](nodes []T, f func(Node) bool) {
	for _, node := range nodes {
		Walk(node, f)
	}
}
