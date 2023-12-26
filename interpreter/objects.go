package interpreter

import (
	"fmt"
	"strconv"

	"github.com/marcuscaisey/golox/token"
)

type loxObject interface {
	String() string
	Type() string
	IsTruthy() loxBool
	UnaryOp(op token.Token) loxObject
	BinaryOp(op token.Token, right loxObject) loxObject
}

func invalidUnaryOpError(op token.Token, object loxObject) error {
	return &runtimeError{
		tok: op,
		msg: fmt.Sprintf("'%s' operator cannot be used with type '%s'", op.Type, object.Type()),
	}
}

func invalidBinaryOpError(op token.Token, left, right loxObject) error {
	return &runtimeError{
		tok: op,
		msg: fmt.Sprintf("'%s' operator cannot be used with types '%s' and '%s'", op.Type, left.Type(), right.Type()),
	}
}

var _ loxObject = loxNumber(0)

type loxNumber float64

func (n loxNumber) String() string {
	return strconv.FormatFloat(float64(n), 'f', -1, 64)
}

func (n loxNumber) Type() string {
	return "number"
}

func (n loxNumber) IsTruthy() loxBool {
	return n != 0
}

func (n loxNumber) UnaryOp(op token.Token) loxObject {
	switch op.Type {
	case token.Minus:
		return -n
	case token.Bang:
		return !n.IsTruthy()
	default:
		panic(invalidUnaryOpError(op, n))
	}
}

func (n loxNumber) BinaryOp(op token.Token, right loxObject) loxObject {
	switch right := right.(type) {
	case loxNumber:
		switch op.Type {
		case token.Equal:
			return loxBool(n == right)
		case token.NotEqual:
			return loxBool(n != right)
		case token.Asterisk:
			return n * right
		case token.Slash:
			return n / right
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
	}
	panic(invalidBinaryOpError(op, n, right))
}

var _ loxObject = loxString("")

type loxString string

func (s loxString) String() string {
	return string(s)
}

func (s loxString) Type() string {
	return "string"
}

func (s loxString) IsTruthy() loxBool {
	return s != ""
}

func (s loxString) UnaryOp(op token.Token) loxObject {
	switch op.Type {
	case token.Bang:
		return !s.IsTruthy()
	default:
		panic(invalidUnaryOpError(op, s))
	}
}

func (s loxString) BinaryOp(op token.Token, right loxObject) loxObject {
	switch right := right.(type) {
	case loxString:
		switch op.Type {
		case token.Equal:
			return loxBool(s == right)
		case token.NotEqual:
			return loxBool(s != right)
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
	}
	panic(invalidBinaryOpError(op, s, right))
}

var _ loxObject = loxBool(false)

type loxBool bool

func (b loxBool) String() string {
	if b {
		return "true"
	}
	return "false"
}

func (b loxBool) Type() string {
	return "bool"
}

func (b loxBool) IsTruthy() loxBool {
	return b
}

func (b loxBool) UnaryOp(op token.Token) loxObject {
	switch op.Type {
	case token.Bang:
		return !b.IsTruthy()
	default:
		panic(invalidUnaryOpError(op, b))
	}
}

func (b loxBool) BinaryOp(op token.Token, right loxObject) loxObject {
	panic(invalidBinaryOpError(op, b, right))
}

var _ loxObject = loxNil{}

type loxNil struct{}

func (n loxNil) String() string {
	return "nil"
}

func (n loxNil) Type() string {
	return "nil"
}

func (n loxNil) IsTruthy() loxBool {
	return false
}

func (n loxNil) UnaryOp(op token.Token) loxObject {
	switch op.Type {
	case token.Bang:
		return !n.IsTruthy()
	default:
		panic(invalidUnaryOpError(op, n))
	}
}

func (n loxNil) BinaryOp(op token.Token, right loxObject) loxObject {
	panic(invalidBinaryOpError(op, n, right))
}
