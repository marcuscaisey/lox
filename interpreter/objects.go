package interpreter

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/fatih/color"

	"github.com/marcuscaisey/golox/token"
)

// Type is the string representation of a Lox object's type.
type Type string

// Format implements fmt.Formatter. All verbs have the default behaviour, except for 'h' (highlight) which prints the
// type in green.
func (t Type) Format(f fmt.State, verb rune) {
	switch verb {
	case 'h':
		fmt.Fprint(f, color.GreenString(string(t)))
	default:
		fmt.Fprintf(f, fmt.FormatString(f, verb), string(t))
	}
}

type loxObject interface {
	String() string
	Type() Type
	IsTruthy() loxBool
	UnaryOp(op token.Token) loxObject
	BinaryOp(op token.Token, right loxObject) loxObject
}

func invalidUnaryOpError(op token.Token, object loxObject) error {
	return &runtimeError{
		tok: op,
		msg: fmt.Sprintf("%h operator cannot be used with type %h", op.Type, object.Type()),
	}
}

func invalidBinaryOpError(op token.Token, left, right loxObject) error {
	return &runtimeError{
		tok: op,
		msg: fmt.Sprintf("%h operator cannot be used with types %h and %h", op.Type, left.Type(), right.Type()),
	}
}

type loxNumber float64

func (n loxNumber) String() string {
	return strconv.FormatFloat(float64(n), 'f', -1, 64)
}

func (n loxNumber) Type() Type {
	return "number"
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
				panic(&runtimeError{
					tok: op,
					msg: "cannot divide by 0",
				})
			}
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
		panic(&runtimeError{
			tok: op,
			msg: "cannot multiply string by non-integer",
		})
	}
	if n < 0 {
		panic(&runtimeError{
			tok: op,
			msg: "cannot multiply string by negative integer",
		})
	}
	return loxString(strings.Repeat(string(s), int(n)))
}

type loxString string

func (s loxString) String() string {
	return string(s)
}

func (s loxString) Type() Type {
	return "string"
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

func (b loxBool) String() string {
	if b {
		return "true"
	}
	return "false"
}

func (b loxBool) Type() Type {
	return "bool"
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

func (n loxNil) String() string {
	return "nil"
}

func (n loxNil) Type() Type {
	return "nil"
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
