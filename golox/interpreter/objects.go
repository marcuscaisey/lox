package interpreter

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/marcuscaisey/lox/lox"
	"github.com/marcuscaisey/lox/lox/ast"
	"github.com/marcuscaisey/lox/lox/token"
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
	CallableName() string
	Params() []string
	Call(interpreter *Interpreter, args []loxObject) loxObject
}

type loxGetter interface {
	Get(interpreter *Interpreter, name token.Token) loxObject
}

type loxSetter interface {
	Set(interpreter *Interpreter, name token.Token, value loxObject)
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
				panic(lox.NewError(op, "cannot divide by 0"))
			}
			return n / right
		case token.Percent:
			if right == 0 {
				panic(lox.NewError(op, "cannot modulo by 0"))
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
		default:
			return nil
		}

	case loxString:
		switch op.Type {
		case token.Asterisk:
			return numberTimesString(n, op, right)
		default:
			return nil
		}

	default:
		return nil
	}
}

func numberTimesString(n loxNumber, op token.Token, s loxString) loxString {
	if math.Floor(float64(n)) != float64(n) {
		panic(lox.NewErrorf(op, "cannot multiply %m by non-integer %m", loxTypeString, loxTypeNumber))
	}
	if n < 0 {
		panic(lox.NewErrorf(op, "cannot multiply %m by negative %m", loxTypeString, loxTypeNumber))
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
		default:
			return nil
		}

	case loxNumber:
		switch op.Type {
		case token.Asterisk:
			return numberTimesString(right, op, s)
		default:
			return nil
		}

	default:
		return nil
	}
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
	funTypeNone     funType = iota
	funTypeFunction funType = 1 << (iota - 1)
	funTypeMethodFlag
	funTypeConstructorFlag
	funTypeBuiltinFlag
)

func (f funType) IsMethod() bool {
	return f&funTypeMethodFlag != 0
}

func (f funType) IsConstructor() bool {
	return f&funTypeConstructorFlag != 0
}

func (f funType) IsBuiltin() bool {
	return f&funTypeBuiltinFlag != 0
}

func methodFunType(decl ast.MethodDecl) funType {
	typ := funTypeFunction | funTypeMethodFlag
	if decl.IsConstructor() {
		typ |= funTypeConstructorFlag
	}
	return typ
}

type nativeFunBody func(args []loxObject) loxObject

type loxFunction struct {
	name       string
	params     []string
	body       []ast.Stmt
	nativeBody nativeFunBody
	typ        funType
	closure    *environment
}

func newLoxFunction(name string, fun ast.Function, typ funType, closure *environment) *loxFunction {
	paramNames := make([]string, len(fun.Params))
	for i, param := range fun.Params {
		paramNames[i] = param.Lexeme
	}
	f := &loxFunction{
		name:    name,
		params:  paramNames,
		body:    fun.Body.Stmts,
		typ:     typ,
		closure: closure,
	}
	return f
}

func newBuiltinLoxFunction(name string, params []string, body nativeFunBody) *loxFunction {
	return &loxFunction{
		name:       name,
		params:     params,
		nativeBody: body,
		typ:        funTypeFunction | funTypeBuiltinFlag,
	}
}

var (
	_ loxObject   = &loxFunction{}
	_ loxCallable = &loxFunction{}
)

func (f *loxFunction) String() string {
	switch {
	case f.typ.IsMethod():
		return fmt.Sprintf("[bound method %s]", f.name)
	case f.typ.IsBuiltin():
		return fmt.Sprintf("[builtin function %s]", f.name)
	default:
		return fmt.Sprintf("[function %s]", f.name)
	}
}

func (f *loxFunction) Type() loxType {
	return loxTypeFunction
}

func (f *loxFunction) CallableName() string {
	return f.name
}

func (f *loxFunction) Params() []string {
	return f.params
}

func (f *loxFunction) Call(interpreter *Interpreter, args []loxObject) loxObject {
	if f.nativeBody != nil {
		return f.nativeBody(args)
	}

	childEnv := f.closure.Child()
	for i, param := range f.params {
		childEnv.Set(param, args[i])
	}
	result := interpreter.executeBlock(childEnv, f.body)
	if f.typ.IsConstructor() {
		return f.closure.GetByIdent(token.CurrentInstanceIdent)
	}
	if r, ok := result.(stmtResultReturn); ok {
		return r.Value
	}
	return loxNil{}
}

func (f *loxFunction) Bind(instance *loxInstance) *loxFunction {
	fCopy := *f
	fCopy.closure = f.closure.Child()
	fCopy.closure.Set(token.CurrentInstanceIdent, instance)
	return &fCopy
}

type property struct {
	getter *loxFunction
	setter *loxFunction
}

func newProperty(getter, setter *loxFunction) *property {
	return &property{getter: getter, setter: setter}
}

func (p *property) Get(interpreter *Interpreter, instance *loxInstance, name token.Token) loxObject {
	return interpreter.call(name.StartPos, p.getter.Bind(instance), nil)
}

func (p *property) Set(interpreter *Interpreter, instance *loxInstance, name token.Token, value loxObject) {
	if p.setter == nil {
		panic(lox.NewErrorf(name, "property '%s' of %m object is read-only", name.Lexeme, instance.Type()))
	}
	interpreter.call(name.StartPos, p.setter.Bind(instance), []loxObject{value})
}

type loxClass struct {
	*loxInstance     // instance of the metaclass or nil for the metaclass itself
	Name             string
	methodsByName    map[string]*loxFunction
	propertiesByName map[string]*property
}

func newLoxClass(name string, methods []ast.MethodDecl, env *environment) *loxClass {
	instanceMethods := make([]ast.MethodDecl, 0, len(methods))
	staticMethods := make([]ast.MethodDecl, 0, len(methods))
	for _, decl := range methods {
		if decl.HasModifier(token.Static) {
			staticMethods = append(staticMethods, decl)
		} else {
			instanceMethods = append(instanceMethods, decl)
		}
	}
	metaclass := newLoxClassWithMetaclass(name, staticMethods, env, nil)
	metaclass.Name = fmt.Sprintf("%s class", name)
	return newLoxClassWithMetaclass(name, instanceMethods, env, metaclass)
}

func newLoxClassWithMetaclass(name string, methods []ast.MethodDecl, env *environment, metaclass *loxClass) *loxClass {
	methodsByName := make(map[string]*loxFunction, len(methods))
	gettersByName := make(map[string]*loxFunction, len(methods))
	settersByName := make(map[string]*loxFunction, len(methods))
	for _, decl := range methods {
		var funcMap map[string]*loxFunction
		methodName := name + "." + decl.Name.Lexeme
		switch {
		case decl.HasModifier(token.Get):
			funcMap = gettersByName
			methodName = "get " + methodName
		case decl.HasModifier(token.Set):
			funcMap = settersByName
			methodName = "set " + methodName
		default:
			funcMap = methodsByName
		}
		funcMap[decl.Name.Lexeme] = newLoxFunction(methodName, decl.Function, methodFunType(decl), env)
	}
	// Every setter must have a corresponding getter so we can just iterate over the getters
	propertiesByName := make(map[string]*property, len(gettersByName))
	for name, getter := range gettersByName {
		propertiesByName[name] = newProperty(getter, settersByName[name])
	}
	class := &loxClass{
		Name:             name,
		methodsByName:    methodsByName,
		propertiesByName: propertiesByName,
	}
	if metaclass != nil {
		class.loxInstance = newLoxInstance(metaclass)
	}
	return class
}

var (
	_ loxObject   = &loxClass{}
	_ loxCallable = &loxClass{}
	_ loxGetter   = &loxClass{}
	_ loxSetter   = &loxClass{}
)

func (c *loxClass) String() string {
	return fmt.Sprintf("[class %s]", c.Name)
}

func (c *loxClass) CallableName() string {
	if init, ok := c.GetMethod(token.ConstructorIdent); ok {
		return init.CallableName()
	}
	return c.Name
}

func (c *loxClass) Params() []string {
	if init, ok := c.GetMethod(token.ConstructorIdent); ok {
		return init.Params()
	}
	return nil
}

func (c *loxClass) Call(interpreter *Interpreter, args []loxObject) loxObject {
	instance := newLoxInstance(c)
	if init, ok := c.GetMethod(token.ConstructorIdent); ok {
		init.Bind(instance).Call(interpreter, args)
	}
	return instance
}

func (c *loxClass) GetMethod(name string) (*loxFunction, bool) {
	method, ok := c.methodsByName[name]
	return method, ok
}

func (c *loxClass) GetProperty(name string) (*property, bool) {
	property, ok := c.propertiesByName[name]
	return property, ok
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

var (
	_ loxObject = &loxInstance{}
	_ loxGetter = &loxInstance{}
	_ loxSetter = &loxInstance{}
)

func (i *loxInstance) String() string {
	return fmt.Sprintf("[%s object]", i.class.Name)
}

func (i *loxInstance) Type() loxType {
	return loxType(i.class.Name)
}

func (i *loxInstance) Get(interpreter *Interpreter, name token.Token) loxObject {
	if property, ok := i.class.GetProperty(name.Lexeme); ok {
		return property.Get(interpreter, i, name)
	}

	if value, ok := i.fieldValuesByName[name.Lexeme]; ok {
		return value
	}

	if method, ok := i.class.GetMethod(name.Lexeme); ok {
		return method.Bind(i)
	}

	panic(lox.NewErrorf(name, "%m object has no property %s", i.Type(), name.Lexeme))
}

func (i *loxInstance) Set(interpreter *Interpreter, name token.Token, value loxObject) {
	if property, ok := i.class.GetProperty(name.Lexeme); ok {
		property.Set(interpreter, i, name, value)
		return
	}

	i.fieldValuesByName[name.Lexeme] = value
}

// errorMsg is a special object which is returned by the built-in error function. It will be caught by the interpreter
// and converted into a runtime error.
type errorMsg string

func (errorMsg) String() string {
	panic("errorMsg is not a real loxObject")
}

func (errorMsg) Type() loxType {
	panic("errorMsg is not a real loxObject")
}
