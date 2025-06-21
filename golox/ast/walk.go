package ast

// Walk traverses an AST in depth-first order: It starts by calling f(node). If f returns true, Walk invokes f
// recursively for each of the non-nil children of node. If node is nil, Walk returns immediately.
func Walk(node Node, f func(Node) bool) {
	if isNil(node) || !f(node) {
		return
	}
	switch node := node.(type) {
	case *Program:
		walkSlice(node.Stmts, f)
	case *Ident:
	case *IllegalStmt:
	case *Comment:
	case *CommentedStmt:
		Walk(node.Stmt, f)
		Walk(node.Comment, f)
	case *VarDecl:
		Walk(node.Name, f)
		Walk(node.Initialiser, f)
	case *FunDecl:
		walkSlice(node.Doc, f)
		Walk(node.Name, f)
		Walk(node.Function, f)
	case *Function:
		walkSlice(node.Params, f)
		Walk(Node(node.Body), f)
	case *ParamDecl:
		Walk(node.Name, f)
	case *ClassDecl:
		walkSlice(node.Doc, f)
		Walk(node.Name, f)
		Walk(node.Body, f)
	case *MethodDecl:
		walkSlice(node.Doc, f)
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
		Walk(node.This, f)
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

func walkSlice[T Node](nodes []T, f func(Node) bool) {
	for _, node := range nodes {
		Walk(node, f)
	}
}

// Find traverses an AST in depth-first order, searching for a non-nil node for which f returns true. If one is found,
// then that node is returned along with true. Otherwise, false and nil are returned.
func Find[T Node](node Node, f func(T) bool) (T, bool) {
	var result T
	var found bool
	Walk(node, func(n Node) bool {
		if nT, ok := n.(T); ok && f(nT) {
			result = nT
			found = true
		}
		return !found
	})
	return result, found
}
