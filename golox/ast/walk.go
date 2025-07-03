package ast

// Walk traverses an AST in depth-first order. If node is nil, Walk returns immediately. Otherwise, it starts by calling
// f(node). If f returns true or node is not of type T, Walk invokes f recursively for each of the non-nil children of
// node.
func Walk[T Node](node Node, f func(T) bool) {
	if nodeT, ok := node.(T); isNil(node) || (ok && !f(nodeT)) {
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

func walkSlice[sliceT, fT Node](nodes []sliceT, f func(fT) bool) {
	for _, node := range nodes {
		Walk(node, f)
	}
}

// Predicate is used by [Find] to determine whether a traversed [Node] should be returned.
type Predicate[T Node] func(T) bool

// Find traverses an AST in depth-first order, searching for a non-nil node for which the predicate p returns true. If
// one is found, then that node is returned along with true. Otherwise, the zero value of T and false are returned.
func Find[T Node](node Node, p Predicate[T]) (T, bool) {
	var result T
	var found bool
	Walk(node, func(n T) bool {
		if p(n) {
			result = n
			found = true
		}
		return !found
	})
	return result, found
}

// FindLast is like [Find] except it returns the last non-nil node that p returns true for instead of the first.
func FindLast[T Node](node Node, p Predicate[T]) (T, bool) {
	var result T
	var found bool
	Walk(node, func(n T) bool {
		if p(n) {
			result = n
			found = true
		}
		return true
	})
	return result, found
}
