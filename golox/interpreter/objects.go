package interpreter

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/lox"
	"github.com/marcuscaisey/lox/golox/token"
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

// Format implements fmt.Formatter. All verbs have the default behaviour, except for 'm' (message) which formats the
// type for use in an error message.
func (t loxType) Format(f fmt.State, verb rune) {
	switch verb {
	case 'm':
		fmt.Fprintf(f, "'%s'", t)
	default:
		fmt.Fprintf(f, fmt.FormatString(f, verb), string(t))
	}
}

type loxObject interface {
	String() string
	Type() loxType
}

type loxUnaryOperand interface {
	// UnaryOp returns the result of applying the given unary operator to the object. If the operator is not supported,
	// then the return value is nil.
	UnaryOp(op token.Token) loxObject
}

type loxBinaryOperand interface {
	// BinaryOp returns the result of applying the given binary operator to the object. If the operator is not
	// supported, then the return value is nil.
	BinaryOp(op token.Token, right loxObject) loxObject
}

type loxTruther interface {
	IsTruthy() loxBool
}

type loxCallable interface {
	Name() string
	Params() []string
	Call(i *Interpreter, args []loxObject) loxObject
}

type loxGetter interface {
	Get(name token.Token) loxObject
}

type loxSetter interface {
	Set(name token.Token, value loxObject)
}

type loxNumber float64

var (
	_ loxObject        = loxNumber(0)
	_ loxUnaryOperand  = loxNumber(0)
	_ loxBinaryOperand = loxNumber(0)
	_ loxTruther       = loxNumber(0)
)

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
	return nil
}

func (n loxNumber) BinaryOp(op token.Token, right loxObject) loxObject {
	switch right := right.(type) {
	case loxNumber:
		switch op.Type {
		case token.Asterisk:
			return n * right
		case token.Slash:
			if right == 0 {
				panic(lox.NewErrorFromToken(op, "cannot divide by 0"))
			}
			return n / right
		case token.Percent:
			if right == 0 {
				panic(lox.NewErrorFromToken(op, "cannot modulo by 0"))
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
	return nil
}

func numberTimesString(n loxNumber, op token.Token, s loxString) loxString {
	if math.Floor(float64(n)) != float64(n) {
		panic(lox.NewErrorFromToken(op, "cannot multiply %m by non-integer %m", loxTypeString, loxTypeNumber))
	}
	if n < 0 {
		panic(lox.NewErrorFromToken(op, "cannot multiply %m by negative %m", loxTypeString, loxTypeNumber))
	}
	return loxString(strings.Repeat(string(s), int(n)))
}

type loxString string

var (
	_ loxObject        = loxString("")
	_ loxBinaryOperand = loxString("")
	_ loxTruther       = loxString("")
)

func (s loxString) String() string {
	return string(s)
}

func (s loxString) Type() loxType {
	return loxTypeString
}

func (s loxString) IsTruthy() loxBool {
	return s != ""
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
	return nil
}

type loxBool bool

var (
	_ loxObject  = loxBool(false)
	_ loxTruther = loxBool(false)
)

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

type loxNil struct{}

var (
	_ loxObject  = loxNil{}
	_ loxTruther = loxNil{}
)

func (n loxNil) String() string {
	return "nil"
}

func (n loxNil) Type() loxType {
	return loxTypeNil
}

func (n loxNil) IsTruthy() loxBool {
	return false
}

type funType int

const (
	funTypeFunction funType = iota
	funTypeMethod
	funTypeInit
)

type loxFunction struct {
	name    string
	params  []string
	body    []ast.Stmt
	typ     funType
	closure *environment
}

func newLoxFunction(name string, params []token.Token, body []ast.Stmt, typ funType, closure *environment) *loxFunction {
	paramNames := make([]string, len(params))
	for i, param := range params {
		paramNames[i] = param.Lexeme
	}
	f := &loxFunction{
		name:    name,
		params:  paramNames,
		body:    body,
		typ:     typ,
		closure: closure,
	}
	return f
}

var (
	_ loxObject   = &loxFunction{}
	_ loxCallable = &loxFunction{}
)

func (f *loxFunction) String() string {
	switch f.typ {
	case funTypeFunction:
		return fmt.Sprintf("[function %s]", f.name)
	case funTypeMethod, funTypeInit:
		return fmt.Sprintf("[bound method %s]", f.name)
	default:
		panic(fmt.Sprintf("unexpected function type %d", f.typ))
	}
}

func (f *loxFunction) Type() loxType {
	return loxTypeFunction
}

func (f *loxFunction) Name() string {
	return f.name
}

func (f *loxFunction) Params() []string {
	return f.params
}

func (f *loxFunction) Call(interpreter *Interpreter, args []loxObject) loxObject {
	childEnv := f.closure.Child()
	for i, param := range f.params {
		childEnv.Set(param, args[i])
	}
	result := interpreter.executeBlock(childEnv, f.body)
	if f.typ == funTypeInit {
		return f.closure.GetByIdent(token.ThisIdent)
	}
	if r, ok := result.(stmtResultReturn); ok {
		return r.Value
	}
	return loxNil{}
}

func (f *loxFunction) Bind(instance *loxInstance) *loxFunction {
	fCopy := *f
	fCopy.closure = f.closure.Child()
	fCopy.closure.Set(token.ThisIdent, instance)
	return &fCopy
}

type loxBuiltinFunction struct {
	name   string
	params []string
	body   func(args []loxObject) loxObject
}

func newLoxBuiltinFunction(name string, params []string, body func(args []loxObject) loxObject) *loxBuiltinFunction {
	return &loxBuiltinFunction{
		name:   name,
		params: params,
		body:   body,
	}
}

var (
	_ loxObject   = &loxBuiltinFunction{}
	_ loxCallable = &loxBuiltinFunction{}
)

func (f *loxBuiltinFunction) String() string {
	return fmt.Sprintf("[builtin function %s]", f.name)
}

func (f *loxBuiltinFunction) Type() loxType {
	return loxTypeFunction
}

func (f *loxBuiltinFunction) Name() string {
	return f.name
}

func (f *loxBuiltinFunction) Params() []string {
	return f.params
}

func (f *loxBuiltinFunction) Call(_ *Interpreter, args []loxObject) loxObject {
	return f.body(args)
}

type loxClass struct {
	*loxInstance  // A class is an instance of its metaclass
	name          string
	init          *loxFunction
	methodsByName map[string]*loxFunction
}

func newLoxClass(name string, instanceMethodsByName map[string]*loxFunction, classMethodsByName map[string]*loxFunction) *loxClass {
	metaclass := &loxClass{
		name:          fmt.Sprintf("%s class", name),
		methodsByName: classMethodsByName,
	}
	class := &loxClass{
		name:          name,
		methodsByName: instanceMethodsByName,
		loxInstance:   metaclass.Call(nil, nil).(*loxInstance),
	}
	if init, ok := class.GetMethod(token.InitIdent); ok {
		class.init = init
	}
	return class
}

var (
	_ loxObject   = &loxClass{}
	_ loxCallable = &loxClass{}
)

func (c *loxClass) String() string {
	return fmt.Sprintf("[class %s]", c.name)
}

func (c *loxClass) Name() string {
	return c.name
}

func (c *loxClass) Params() []string {
	if c.init == nil {
		return nil
	}
	return c.init.Params()
}

func (c *loxClass) Call(i *Interpreter, args []loxObject) loxObject {
	instance := newLoxInstance(c)
	if c.init != nil {
		c.init.Bind(instance).Call(i, args)
	}
	return instance
}

func (c *loxClass) GetMethod(name string) (*loxFunction, bool) {
	method, ok := c.methodsByName[name]
	return method, ok
}

type loxInstance struct {
	class             *loxClass
	fieldValuesByName map[string]loxObject
}

func newLoxInstance(class *loxClass) *loxInstance {
	return &loxInstance{
		class:             class,
		fieldValuesByName: make(map[string]loxObject),
	}
}

var _ loxObject = &loxInstance{}

func (i *loxInstance) String() string {
	return fmt.Sprintf("[%s object]", i.class.Name())
}

func (i *loxInstance) Type() loxType {
	return loxType(i.class.Name())
}

func (i *loxInstance) Get(name token.Token) loxObject {
	if value, ok := i.fieldValuesByName[name.Lexeme]; ok {
		return value
	}

	if method, ok := i.class.GetMethod(name.Lexeme); ok {
		return method.Bind(i)
	}

	panic(lox.NewErrorFromToken(name, "%m object has no property %s", i.Type(), name.Lexeme))
}

func (i *loxInstance) Set(name token.Token, value loxObject) {
	i.fieldValuesByName[name.Lexeme] = value
}
