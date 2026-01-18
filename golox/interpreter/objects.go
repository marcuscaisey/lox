package interpreter

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/marcuscaisey/lox/golox/ast"
	"github.com/marcuscaisey/lox/golox/loxerr"
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
	CallableName() string
	Params() []string
	Call(interpreter *Interpreter, args []loxObject) loxObject
}

type loxGetter interface {
	Get(interpreter *Interpreter, name *ast.Ident) loxObject
}

type loxSetter interface {
	Set(interpreter *Interpreter, name *ast.Ident, value loxObject)
}

type loxNumber float64

var (
	_ loxObject        = loxNumber(0)
	_ loxUnaryOperand  = loxNumber(0)
	_ loxBinaryOperand = loxNumber(0)
)

func (n loxNumber) String() string {
	return strconv.FormatFloat(float64(n), 'f', -1, 64)
}

func (n loxNumber) Type() loxType {
	return loxTypeNumber
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
				panic(loxerr.Newf(op, loxerr.Fatal, "cannot divide by 0"))
			}
			return n / right
		case token.Percent:
			if right == 0 {
				panic(loxerr.Newf(op, loxerr.Fatal, "cannot modulo by 0"))
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
		panic(loxerr.Newf(op, loxerr.Fatal, "cannot multiply %m by non-integer %m", loxTypeString, loxTypeNumber))
	}
	if n < 0 {
		panic(loxerr.Newf(op, loxerr.Fatal, "cannot multiply %m by negative %m", loxTypeString, loxTypeNumber))
	}
	return loxString(strings.Repeat(string(s), int(n)))
}

type loxString string

var (
	_ loxObject        = loxString("")
	_ loxBinaryOperand = loxString("")
)

func (s loxString) String() string {
	return string(s)
}

func (s loxString) Type() loxType {
	return loxTypeString
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

func methodFunType(decl *ast.MethodDecl) funType {
	typ := funTypeFunction | funTypeMethodFlag
	if decl.IsConstructor() {
		typ |= funTypeConstructorFlag
	}
	return typ
}

type nativeFunBody func(args []loxObject) loxObject

type loxFunction struct {
	name         string
	params       []string
	body         []ast.Stmt
	nativeBody   nativeFunBody
	typ          funType
	enclosingEnv environment
}

func newLoxFunction(name string, fun *ast.Function, typ funType, closure environment) *loxFunction {
	paramNames := make([]string, len(fun.Params))
	for i, param := range fun.Params {
		paramNames[i] = param.Name.String()
	}
	f := &loxFunction{
		name:         name,
		params:       paramNames,
		body:         fun.Body.Stmts,
		typ:          typ,
		enclosingEnv: closure,
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

	childEnv := f.enclosingEnv.Child()
	for i, param := range f.params {
		childEnv = childEnv.Define(param, args[i])
	}
	result := interpreter.executeBlock(childEnv, f.body)
	if f.typ.IsConstructor() {
		return f.enclosingEnv.Get(&ast.Ident{Token: token.Token{Lexeme: token.IdentThis}})
	}
	if r, ok := result.(stmtResultReturn); ok {
		return r.Value
	}
	return loxNil{}
}

func (f *loxFunction) Bind(instance *loxInstance) *loxFunction {
	fCopy := *f
	fCopyClosure := f.enclosingEnv.Child()
	fCopy.enclosingEnv = fCopyClosure.Define(token.IdentThis, instance)
	return &fCopy
}

type property struct {
	getter *loxFunction
	setter *loxFunction
}

func newProperty(getter, setter *loxFunction) *property {
	return &property{getter: getter, setter: setter}
}

func (p *property) Get(interpreter *Interpreter, instance *loxInstance, name *ast.Ident) loxObject {
	return interpreter.call(name.Start(), p.getter.Bind(instance), nil)
}

func (p *property) Set(interpreter *Interpreter, instance *loxInstance, name *ast.Ident, value loxObject) {
	if p.setter == nil {
		panic(loxerr.Newf(name, loxerr.Fatal, "property '%s' of %m object is read-only", name.String(), instance.Type()))
	}
	interpreter.call(name.Start(), p.setter.Bind(instance), []loxObject{value})
}

type loxClass struct {
	Name              string
	superclass        *loxClass
	metaclassInstance *loxInstance
	methodsByName     map[string]*loxFunction
	propertiesByName  map[string]*property
}

func newLoxClass(name string, superclass *loxClass, methods []*ast.MethodDecl, env environment) *loxClass {
	instanceMethods := make([]*ast.MethodDecl, 0, len(methods))
	staticMethods := make([]*ast.MethodDecl, 0, len(methods))
	for _, decl := range methods {
		if decl.HasModifier(token.Static) {
			staticMethods = append(staticMethods, decl)
		} else {
			instanceMethods = append(instanceMethods, decl)
		}
	}
	var metaclassSuperclass *loxClass
	if superclass != nil && superclass.metaclassInstance != nil {
		metaclassSuperclass = superclass.metaclassInstance.Class
	}
	metaclass := newLoxClassWithMetaclass(name, metaclassSuperclass, nil, staticMethods, env)
	metaclass.Name = fmt.Sprintf("%s class", name)
	return newLoxClassWithMetaclass(name, superclass, metaclass, instanceMethods, env)
}

func newLoxClassWithMetaclass(name string, superclass *loxClass, metaclass *loxClass, methods []*ast.MethodDecl, env environment) *loxClass {
	methodsByName := make(map[string]*loxFunction, len(methods))
	gettersByName := make(map[string]*loxFunction, len(methods))
	settersByName := make(map[string]*loxFunction, len(methods))
	if superclass != nil {
		env = env.Child()
		env = env.Define(token.IdentSuper, superclass)
	}
	for _, decl := range methods {
		var funcMap map[string]*loxFunction
		methodName := name + "." + decl.Name.String()
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
		funcMap[decl.Name.String()] = newLoxFunction(methodName, decl.Function, methodFunType(decl), env)
	}
	// Every setter must have a corresponding getter so we can just iterate over the getters
	propertiesByName := make(map[string]*property, len(gettersByName))
	for name, getter := range gettersByName {
		propertiesByName[name] = newProperty(getter, settersByName[name])
	}
	class := &loxClass{
		Name:             name,
		superclass:       superclass,
		methodsByName:    methodsByName,
		propertiesByName: propertiesByName,
	}
	if metaclass != nil {
		class.metaclassInstance = newLoxInstance(metaclass)
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

func (c *loxClass) Type() loxType {
	return c.metaclassInstance.Type()
}

func (c *loxClass) CallableName() string {
	if init, ok := c.GetMethod(token.IdentInit); ok {
		return init.CallableName()
	}
	return c.Name
}

func (c *loxClass) Params() []string {
	if init, ok := c.GetMethod(token.IdentInit); ok {
		return init.Params()
	}
	return nil
}

func (c *loxClass) Call(interpreter *Interpreter, args []loxObject) loxObject {
	instance := newLoxInstance(c)
	if init, ok := c.GetMethod(token.IdentInit); ok {
		init.Bind(instance).Call(interpreter, args)
	}
	return instance
}

func (c *loxClass) Set(interpreter *Interpreter, name *ast.Ident, value loxObject) {
	c.metaclassInstance.Set(interpreter, name, value)
}

func (c *loxClass) Get(interpreter *Interpreter, name *ast.Ident) loxObject {
	return c.metaclassInstance.Get(interpreter, name)
}

func (c *loxClass) GetMethod(name string) (*loxFunction, bool) {
	if method, ok := c.methodsByName[name]; ok {
		return method, true
	}
	if c.superclass != nil {
		return c.superclass.GetMethod(name)
	}
	return nil, false
}

func (c *loxClass) GetProperty(name string) (*property, bool) {
	if property, ok := c.propertiesByName[name]; ok {
		return property, true
	}
	if c.superclass != nil {
		return c.superclass.GetProperty(name)
	}
	return nil, false
}

type loxSuperObject struct {
	superclass   *loxClass
	enclosingEnv environment
}

func newLoxSuperObject(superclass *loxClass, enclosingEnv environment) *loxSuperObject {
	return &loxSuperObject{
		superclass:   superclass,
		enclosingEnv: enclosingEnv,
	}
}

var (
	_ loxObject = &loxSuperObject{}
	_ loxGetter = &loxSuperObject{}
)

func (s *loxSuperObject) Get(_ *Interpreter, name *ast.Ident) loxObject {
	instanceObject := s.enclosingEnv.GetByName(token.IdentThis)
	instance, ok := instanceObject.(*loxInstance)
	if !ok {
		panic(fmt.Sprintf("unexpected instance type: %T", instanceObject))
	}
	method, ok := s.superclass.GetMethod(name.String())
	if !ok {
		panic(loxerr.Newf(name, loxerr.Fatal, "%m object has no property %s", instance.Type(), name))
	}
	return method.Bind(instance)
}

func (s *loxSuperObject) String() string {
	return s.superclass.String()
}

func (s *loxSuperObject) Type() loxType {
	return s.superclass.Type()
}

type loxInstance struct {
	Class             *loxClass
	fieldValuesByName map[string]loxObject
}

func newLoxInstance(class *loxClass) *loxInstance {
	return &loxInstance{
		Class:             class,
		fieldValuesByName: make(map[string]loxObject),
	}
}

var (
	_ loxObject = &loxInstance{}
	_ loxGetter = &loxInstance{}
	_ loxSetter = &loxInstance{}
)

func (i *loxInstance) String() string {
	return fmt.Sprintf("[%s object]", i.Class.Name)
}

func (i *loxInstance) Type() loxType {
	return loxType(i.Class.Name)
}

func (i *loxInstance) Get(interpreter *Interpreter, name *ast.Ident) loxObject {
	if property, ok := i.Class.GetProperty(name.String()); ok {
		return property.Get(interpreter, i, name)
	}

	if value, ok := i.fieldValuesByName[name.String()]; ok {
		return value
	}

	if method, ok := i.Class.GetMethod(name.String()); ok {
		return method.Bind(i)
	}

	panic(loxerr.Newf(name, loxerr.Fatal, "%m object has no property %s", i.Type(), name.String()))
}

func (i *loxInstance) Set(interpreter *Interpreter, name *ast.Ident, value loxObject) {
	if property, ok := i.Class.GetProperty(name.String()); ok {
		property.Set(interpreter, i, name, value)
		return
	}

	i.fieldValuesByName[name.String()] = value
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
