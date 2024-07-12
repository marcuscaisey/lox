package interpreter

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/fatih/color"

	"github.com/marcuscaisey/golox/ast"
	"github.com/marcuscaisey/golox/token"
)

// loxType is the string representation of a Lox object's type.
type loxType string

const (
	loxTypeNumber   loxType = "number"
	loxTypeString   loxType = "string"
	loxTypeBool     loxType = "bool"
	loxTypeNil      loxType = "nil"
	loxTypeFunction loxType = "function"
)

// Format implements fmt.Formatter. All verbs have the default behaviour, except for 'h' (highlight) which prints the
// type in green.
func (t loxType) Format(f fmt.State, verb rune) {
	switch verb {
	case 'h':
		fmt.Fprint(f, color.GreenString(string(t)))
	default:
		fmt.Fprintf(f, fmt.FormatString(f, verb), string(t))
	}
}

type loxObject interface {
	String() string
	Type() loxType
	IsTruthy() loxBool
	// TODO: Extract these out to separate interface(s)?
	UnaryOp(op token.Token) loxObject
	BinaryOp(op token.Token, right loxObject) loxObject
}

type loxCallable interface {
	loxObject
	Name() string
	Params() []string
	Call(i *Interpreter, env *environment, args []loxObject) loxObject
}

func invalidUnaryOpError(op token.Token, object loxObject) error {
	return newTokenRuntimeErrorf(op, "%h operator cannot be used with type %h", op.Type, object.Type())
}

func invalidBinaryOpError(op token.Token, left, right loxObject) error {
	return newTokenRuntimeErrorf(op, "%h operator cannot be used with types %h and %h", op.Type, left.Type(), right.Type())
}

type loxNumber float64

var _ loxObject = loxNumber(0)

func (n loxNumber) String() string {
	return strconv.FormatFloat(float64(n), 'f', -1, 64)
}

func (n loxNumber) Type() loxType {
	return loxTypeNumber
}

func (n loxNumber) IsTruthy() loxBool {
	return n != 0
}

func (n loxNumber) UnaryOp(op token.Token) loxObject {
	if op.Type == token.Minus {
		return -n
	}
	panic(invalidUnaryOpError(op, n))
}

func (n loxNumber) BinaryOp(op token.Token, right loxObject) loxObject {
	switch right := right.(type) {
	case loxNumber:
		switch op.Type {
		case token.Asterisk:
			return n * right
		case token.Slash:
			if right == 0 {
				panic(newTokenRuntimeErrorf(op, "cannot divide by 0"))
			}
			return n / right
		case token.Percent:
			if right == 0 {
				panic(newTokenRuntimeErrorf(op, "cannot modulo by 0"))
			}
			return loxNumber(math.Mod(float64(n), float64(right)))
		case token.Plus:
			return n + right
		case token.Minus:
			return n - right
		case token.Less:
			return loxBool(n < right)
		case token.LessEqual:
			return loxBool(n <= right)
		case token.Greater:
			return loxBool(n > right)
		case token.GreaterEqual:
			return loxBool(n >= right)
		}
	case loxString:
		switch op.Type {
		case token.Asterisk:
			return numberTimesString(n, op, right)
		}
	}
	panic(invalidBinaryOpError(op, n, right))
}

func numberTimesString(n loxNumber, op token.Token, s loxString) loxString {
	if math.Floor(float64(n)) != float64(n) {
		panic(newTokenRuntimeErrorf(op, "cannot multiply %h by non-integer %h", loxTypeString, loxTypeNumber))
	}
	if n < 0 {
		panic(newTokenRuntimeErrorf(op, "cannot multiply %h by negative %h", loxTypeString, loxTypeNumber))
	}
	return loxString(strings.Repeat(string(s), int(n)))
}

type loxString string

var _ loxObject = loxString("")

func (s loxString) String() string {
	return string(s)
}

func (s loxString) Type() loxType {
	return loxTypeString
}

func (s loxString) IsTruthy() loxBool {
	return s != ""
}

func (s loxString) UnaryOp(op token.Token) loxObject {
	panic(invalidUnaryOpError(op, s))
}

func (s loxString) BinaryOp(op token.Token, right loxObject) loxObject {
	switch right := right.(type) {
	case loxString:
		switch op.Type {
		case token.Plus:
			return s + right
		case token.Less:
			return loxBool(s < right)
		case token.LessEqual:
			return loxBool(s <= right)
		case token.Greater:
			return loxBool(s > right)
		case token.GreaterEqual:
			return loxBool(s >= right)
		}
	case loxNumber:
		switch op.Type {
		case token.Asterisk:
			return numberTimesString(right, op, s)
		}
	}
	panic(invalidBinaryOpError(op, s, right))
}

type loxBool bool

var _ loxObject = loxBool(false)

func (b loxBool) String() string {
	if b {
		return "true"
	}
	return "false"
}

func (b loxBool) Type() loxType {
	return loxTypeBool
}

func (b loxBool) IsTruthy() loxBool {
	return b
}

func (b loxBool) UnaryOp(op token.Token) loxObject {
	panic(invalidUnaryOpError(op, b))
}

func (b loxBool) BinaryOp(op token.Token, right loxObject) loxObject {
	panic(invalidBinaryOpError(op, b, right))
}

type loxNil struct{}

var _ loxObject = loxNil{}

func (n loxNil) String() string {
	return "nil"
}

func (n loxNil) Type() loxType {
	return loxTypeNil
}

func (n loxNil) IsTruthy() loxBool {
	return false
}

func (n loxNil) UnaryOp(op token.Token) loxObject {
	panic(invalidUnaryOpError(op, n))
}

func (n loxNil) BinaryOp(op token.Token, right loxObject) loxObject {
	panic(invalidBinaryOpError(op, n, right))
}

type loxFunction struct {
	name    string
	params  []token.Token
	body    []ast.Stmt
	closure *environment
}

var _ loxCallable = loxFunction{}

func (f loxFunction) String() string {
	return fmt.Sprintf("<function %s>", f.name)
}

func (f loxFunction) Type() loxType {
	return loxTypeFunction
}

func (f loxFunction) IsTruthy() loxBool {
	return true
}

func (f loxFunction) UnaryOp(op token.Token) loxObject {
	panic(invalidUnaryOpError(op, f))
}

func (f loxFunction) BinaryOp(op token.Token, right loxObject) loxObject {
	panic(invalidBinaryOpError(op, f, right))
}

func (f loxFunction) Name() string {
	return f.name
}

func (f loxFunction) Params() []string {
	params := make([]string, len(f.params))
	for i, param := range f.params {
		params[i] = param.Literal
	}
	return params
}

func (f loxFunction) Call(interpreter *Interpreter, env *environment, args []loxObject) loxObject {
	childEnv := f.closure.Child()
	for i, param := range f.Params() {
		childEnv.Set(param, args[i])
	}
	result := interpreter.executeBlock(childEnv, f.body)
	if r, ok := result.(stmtResultReturn); ok {
		return r.Value
	}
	return loxNil{}
}

type loxBuiltinFunction struct {
	name   string
	params []string
	fn     func(args []loxObject) loxObject
}

var _ loxCallable = loxBuiltinFunction{}

func (f loxBuiltinFunction) String() string {
	return fmt.Sprintf("<builtin function %s>", f.name)
}

func (f loxBuiltinFunction) Type() loxType {
	return loxTypeFunction
}

func (f loxBuiltinFunction) IsTruthy() loxBool {
	return true
}

func (f loxBuiltinFunction) UnaryOp(op token.Token) loxObject {
	panic(invalidUnaryOpError(op, f))
}

func (f loxBuiltinFunction) BinaryOp(op token.Token, right loxObject) loxObject {
	panic(invalidBinaryOpError(op, f, right))
}

func (f loxBuiltinFunction) Name() string {
	return f.name
}

func (f loxBuiltinFunction) Params() []string {
	return f.params
}

func (f loxBuiltinFunction) Call(_ *Interpreter, _ *environment, args []loxObject) loxObject {
	return f.fn(args)
}
