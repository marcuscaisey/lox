package interpreter

import (
	"fmt"
	"math"
	"slices"
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
	loxTypeList     loxType = "list"
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
	Repr() string
	Type() loxType
	Equals(other loxObject) bool
}

type loxUnaryOperand interface {
	UnaryOp(op token.Token) loxObject
}

func newInvalidUnaryOpError(op token.Token, right loxObject) error {
	return loxerr.Newf(op, loxerr.Fatal, "%m operator cannot be used with type %m", op.Type, right.Type())
}

type loxBinaryOperand interface {
	BinaryOp(op token.Token, right loxObject) loxObject
}

func newInvalidBinaryOpError(op token.Token, left loxObject, right loxObject) error {
	return loxerr.Newf(op, loxerr.Fatal, "%m operator cannot be used with types %m and %m", op.Type, left.Type(), right.Type())
}

type loxTruther interface {
	IsTruthy() loxBool
}

type loxCallable interface {
	CallableName() string
	Params() []string
	Call(interpreter *Interpreter, args []loxObject) loxObject
}

type loxIndexable interface {
	Index(index loxObject, node ast.Node) loxObject
	SetIndex(index loxObject, node ast.Node, value loxObject)
}

type loxPropertyAccessible interface {
	Property(interpreter *Interpreter, name *ast.Ident) loxObject
}

type loxPropertySettable interface {
	SetProperty(interpreter *Interpreter, name *ast.Ident, value loxObject)
}

type loxNumber float64

var (
	_ loxObject        = loxNumber(0)
	_ loxUnaryOperand  = loxNumber(0)
	_ loxBinaryOperand = loxNumber(0)
)

func (l loxNumber) String() string {
	return strconv.FormatFloat(float64(l), 'f', -1, 64)
}

func (l loxNumber) Repr() string {
	return l.String()
}

func (l loxNumber) Type() loxType {
	return loxTypeNumber
}

func (l loxNumber) Equals(other loxObject) bool {
	otherNumber, ok := other.(loxNumber)
	return ok && l == otherNumber
}

func (l loxNumber) UnaryOp(op token.Token) loxObject {
	if op.Type == token.Minus {
		return -l
	}
	panic(newInvalidUnaryOpError(op, l))
}

func (l loxNumber) BinaryOp(op token.Token, right loxObject) loxObject {
rightSwitch:
	switch right := right.(type) {
	case loxNumber:
		switch op.Type {
		case token.Asterisk:
			return l * right
		case token.Slash:
			if right == 0 {
				panic(loxerr.Newf(op, loxerr.Fatal, "cannot divide by 0"))
			}
			return l / right
		case token.Percent:
			if right == 0 {
				panic(loxerr.Newf(op, loxerr.Fatal, "cannot modulo by 0"))
			}
			return loxNumber(math.Mod(float64(l), float64(right)))
		case token.Plus:
			return l + right
		case token.Minus:
			return l - right
		case token.Less:
			return loxBool(l < right)
		case token.LessEqual:
			return loxBool(l <= right)
		case token.Greater:
			return loxBool(l > right)
		case token.GreaterEqual:
			return loxBool(l >= right)
		default:
			break rightSwitch
		}

	case loxString:
		switch op.Type {
		case token.Asterisk:
			return numberTimesString(l, op, right)
		default:
			break rightSwitch
		}

	case *loxList:
		switch op.Type {
		case token.Asterisk:
			return numberTimesList(l, op, right)
		default:
			break rightSwitch
		}

	default:
		break rightSwitch
	}

	panic(newInvalidBinaryOpError(op, l, right))
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

func numberTimesList(n loxNumber, op token.Token, l *loxList) *loxList {
	if math.Floor(float64(n)) != float64(n) {
		panic(loxerr.Newf(op, loxerr.Fatal, "cannot multiply %m by non-integer %m", loxTypeList, loxTypeNumber))
	}
	if n < 0 {
		panic(loxerr.Newf(op, loxerr.Fatal, "cannot multiply %m by negative %m", loxTypeList, loxTypeNumber))
	}
	result := loxList(slices.Repeat(*l, int(n)))
	return &result
}

type loxString string

var (
	_ loxObject        = loxString("")
	_ loxBinaryOperand = loxString("")
)

func (s loxString) String() string {
	return string(s)
}

func (s loxString) Repr() string {
	return fmt.Sprintf("%q", s)
}

func (s loxString) Type() loxType {
	return loxTypeString
}

func (s loxString) Equals(other loxObject) bool {
	otherString, ok := other.(loxString)
	return ok && s == otherString
}

func (s loxString) BinaryOp(op token.Token, right loxObject) loxObject {
rightSwitch:
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
			break rightSwitch
		}

	case loxNumber:
		switch op.Type {
		case token.Asterisk:
			return numberTimesString(right, op, s)
		default:
			break rightSwitch
		}

	default:
		break rightSwitch
	}

	panic(newInvalidBinaryOpError(op, s, right))
}

type loxBool bool

var (
	_ loxObject  = loxBool(false)
	_ loxTruther = loxBool(false)
)

func (b loxBool) String() string {
	if b {
		return token.True.String()
	}
	return token.False.String()
}

func (b loxBool) Repr() string {
	return b.String()
}

func (b loxBool) Type() loxType {
	return loxTypeBool
}

func (b loxBool) Equals(other loxObject) bool {
	otherBool, ok := other.(loxBool)
	return ok && b == otherBool
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
	return token.Nil.String()
}

func (n loxNil) Repr() string {
	return n.String()
}

func (n loxNil) Type() loxType {
	return loxTypeNil
}

func (n loxNil) Equals(other loxObject) bool {
	otherNil, ok := other.(loxNil)
	return ok && n == otherNil
}

func (n loxNil) IsTruthy() loxBool {
	return false
}

type funType int

const (
	funTypeNone     funType = iota
	funTypeFunction funType = 1 << (iota - 1)
	funTypeMethodFlag
	funTypeInitFlag
	funTypeBuiltInFlag
)

func (f funType) IsMethod() bool {
	return f&funTypeMethodFlag != 0
}

func (f funType) IsInit() bool {
	return f&funTypeInitFlag != 0
}

func (f funType) IsBuiltIn() bool {
	return f&funTypeBuiltInFlag != 0
}

func methodFunType(decl *ast.MethodDecl) funType {
	typ := funTypeFunction | funTypeMethodFlag
	if decl.IsInit() {
		typ |= funTypeInitFlag
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

func newBuiltInLoxFunction(name string, params []string, body nativeFunBody) *loxFunction {
	return &loxFunction{
		name:       name,
		params:     params,
		nativeBody: body,
		typ:        funTypeFunction | funTypeBuiltInFlag,
	}
}

func newBuiltInLoxMethod(name string, params []string, body nativeFunBody) *loxFunction {
	return &loxFunction{
		name:       name,
		params:     params,
		nativeBody: body,
		typ:        funTypeMethodFlag | funTypeBuiltInFlag,
	}
}

var (
	_ loxObject   = &loxFunction{}
	_ loxCallable = &loxFunction{}
)

func (f *loxFunction) String() string {
	switch {
	case f.typ.IsMethod() && f.typ.IsBuiltIn():
		return fmt.Sprintf("[built-in method %s]", f.name)
	case f.typ.IsMethod():
		return fmt.Sprintf("[bound method %s]", f.name)
	case f.typ.IsBuiltIn():
		return fmt.Sprintf("[built-in function %s]", f.name)
	default:
		return fmt.Sprintf("[function %s]", f.name)
	}
}

func (f *loxFunction) Repr() string {
	return f.String()
}

func (f *loxFunction) Type() loxType {
	return loxTypeFunction
}

func (f *loxFunction) Equals(other loxObject) bool {
	otherFunction, ok := other.(*loxFunction)
	return ok && f == otherFunction
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
	if f.typ.IsInit() {
		return f.enclosingEnv.GetByName(token.This.String())
	}
	if r, ok := result.(stmtResultReturn); ok {
		return r.Value
	}
	return loxNil{}
}

func (f *loxFunction) Bind(instance *loxInstance) *loxFunction {
	fCopy := *f
	fCopyClosure := f.enclosingEnv.Child()
	fCopy.enclosingEnv = fCopyClosure.Define(token.This.String(), instance)
	return &fCopy
}

type propertyAccessors struct {
	getter *loxFunction
	setter *loxFunction
}

func newPropertyAccessors(getter, setter *loxFunction) *propertyAccessors {
	return &propertyAccessors{getter: getter, setter: setter}
}

func (p *propertyAccessors) Get(interpreter *Interpreter, instance *loxInstance, name *ast.Ident) loxObject {
	return interpreter.call(name.Start(), p.getter.Bind(instance), nil)
}

func (p *propertyAccessors) Set(interpreter *Interpreter, instance *loxInstance, name *ast.Ident, value loxObject) {
	if p.setter == nil {
		panic(loxerr.Newf(name, loxerr.Fatal, "property '%s' of %m object is read-only", name.String(), instance.Type()))
	}
	interpreter.call(name.Start(), p.setter.Bind(instance), []loxObject{value})
}

const metaclassNameSuffix = " class"

type loxClass struct {
	Name                    string
	superclass              *loxClass
	metaclassInstance       *loxInstance
	methodsByName           map[string]*loxFunction
	propertyAccessorsByName map[string]*propertyAccessors
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
	metaclass.Name = fmt.Sprint(name, metaclassNameSuffix)
	return newLoxClassWithMetaclass(name, superclass, metaclass, instanceMethods, env)
}

func newLoxClassWithMetaclass(name string, superclass *loxClass, metaclass *loxClass, methods []*ast.MethodDecl, env environment) *loxClass {
	methodsByName := make(map[string]*loxFunction, len(methods))
	gettersByName := make(map[string]*loxFunction, len(methods))
	settersByName := make(map[string]*loxFunction, len(methods))
	if superclass != nil {
		env = env.Child()
		env = env.Define(token.Super.String(), superclass)
	}
	for _, decl := range methods {
		var funcMap map[string]*loxFunction
		methodName := fmt.Sprint(name, token.Dot, decl.Name.String())
		switch {
		case decl.HasModifier(token.Get):
			funcMap = gettersByName
			methodName = fmt.Sprint(token.Get, " ", methodName)
		case decl.HasModifier(token.Set):
			funcMap = settersByName
			methodName = fmt.Sprint(token.Set, " ", methodName)
		default:
			funcMap = methodsByName
		}
		funcMap[decl.Name.String()] = newLoxFunction(methodName, decl.Function, methodFunType(decl), env)
	}
	// Every setter must have a corresponding getter so we can just iterate over the getters
	propertyAccessorsByName := make(map[string]*propertyAccessors, len(gettersByName))
	for name, getter := range gettersByName {
		propertyAccessorsByName[name] = newPropertyAccessors(getter, settersByName[name])
	}
	class := &loxClass{
		Name:                    name,
		superclass:              superclass,
		methodsByName:           methodsByName,
		propertyAccessorsByName: propertyAccessorsByName,
	}
	if metaclass != nil {
		class.metaclassInstance = newLoxInstance(metaclass)
	}
	return class
}

var (
	_ loxObject             = &loxClass{}
	_ loxCallable           = &loxClass{}
	_ loxPropertyAccessible = &loxClass{}
	_ loxPropertySettable   = &loxClass{}
)

func (c *loxClass) String() string {
	return fmt.Sprintf("[class %s]", c.Name)
}

func (c *loxClass) Repr() string {
	return c.String()
}

func (c *loxClass) Type() loxType {
	return c.metaclassInstance.Type()
}

func (c *loxClass) Equals(other loxObject) bool {
	otherClass, ok := other.(*loxClass)
	return ok && c == otherClass
}

func (c *loxClass) CallableName() string {
	if init, ok := c.Method(token.IdentInit); ok {
		return init.CallableName()
	}
	return c.Name
}

func (c *loxClass) Params() []string {
	if init, ok := c.Method(token.IdentInit); ok {
		return init.Params()
	}
	return nil
}

func (c *loxClass) Call(interpreter *Interpreter, args []loxObject) loxObject {
	instance := newLoxInstance(c)
	if init, ok := c.Method(token.IdentInit); ok {
		init.Bind(instance).Call(interpreter, args)
	}
	return instance
}

func (c *loxClass) SetProperty(interpreter *Interpreter, name *ast.Ident, value loxObject) {
	c.metaclassInstance.SetProperty(interpreter, name, value)
}

func (c *loxClass) Property(interpreter *Interpreter, name *ast.Ident) loxObject {
	return c.metaclassInstance.Property(interpreter, name)
}

func (c *loxClass) Method(name string) (*loxFunction, bool) {
	if method, ok := c.methodsByName[name]; ok {
		return method, true
	}
	if c.superclass != nil {
		return c.superclass.Method(name)
	}
	return nil, false
}

func (c *loxClass) PropertyAccessors(name string) (*propertyAccessors, bool) {
	if propertyAccessors, ok := c.propertyAccessorsByName[name]; ok {
		return propertyAccessors, true
	}
	if c.superclass != nil {
		return c.superclass.PropertyAccessors(name)
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
	_ loxObject             = &loxSuperObject{}
	_ loxPropertyAccessible = &loxSuperObject{}
)

func (s *loxSuperObject) Property(_ *Interpreter, name *ast.Ident) loxObject {
	instanceObject := s.enclosingEnv.GetByName(token.This.String())
	instance, ok := instanceObject.(*loxInstance)
	if !ok {
		panic(fmt.Sprintf("unexpected instance type: %T", instanceObject))
	}
	method, ok := s.superclass.Method(name.String())
	if !ok {
		static := ""
		superclassName, ok := strings.CutSuffix(s.superclass.Name, metaclassNameSuffix)
		if ok {
			static = "static "
		}
		panic(loxerr.Newf(name, loxerr.Fatal, "'%s' class has no %smethod %m", superclassName, static, name))
	}
	return method.Bind(instance)
}

func (s *loxSuperObject) String() string {
	return s.superclass.String()
}

func (s *loxSuperObject) Repr() string {
	return s.String()
}

func (s *loxSuperObject) Type() loxType {
	return s.superclass.Type()
}

func (s *loxSuperObject) Equals(other loxObject) bool {
	otherSuper, ok := other.(*loxSuperObject)
	return ok && s == otherSuper
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
	_ loxObject             = &loxInstance{}
	_ loxPropertyAccessible = &loxInstance{}
	_ loxPropertySettable   = &loxInstance{}
)

func (i *loxInstance) String() string {
	return fmt.Sprintf("[%s object]", i.Class.Name)
}

func (i *loxInstance) Repr() string {
	return i.String()
}

func (i *loxInstance) Type() loxType {
	return loxType(i.Class.Name)
}

func (i *loxInstance) Equals(other loxObject) bool {
	otherInstance, ok := other.(*loxInstance)
	return ok && i == otherInstance
}

func (i *loxInstance) Property(interpreter *Interpreter, name *ast.Ident) loxObject {
	if propertyAccessors, ok := i.Class.PropertyAccessors(name.String()); ok {
		return propertyAccessors.Get(interpreter, i, name)
	}

	if value, ok := i.fieldValuesByName[name.String()]; ok {
		return value
	}

	if method, ok := i.Class.Method(name.String()); ok {
		return method.Bind(i)
	}

	panic(loxerr.Newf(name, loxerr.Fatal, "%m object has no property %m", i.Type(), name))
}

func (i *loxInstance) SetProperty(interpreter *Interpreter, name *ast.Ident, value loxObject) {
	if propertyAccessors, ok := i.Class.PropertyAccessors(name.String()); ok {
		propertyAccessors.Set(interpreter, i, name, value)
		return
	}

	i.fieldValuesByName[name.String()] = value
}

type loxList []loxObject

var (
	_ loxObject             = (*loxList)(nil)
	_ loxBinaryOperand      = (*loxList)(nil)
	_ loxIndexable          = (*loxList)(nil)
	_ loxPropertyAccessible = (*loxList)(nil)
)

func (l *loxList) String() string {
	b := new(strings.Builder)
	fmt.Fprint(b, token.LeftBrack)
	for i, el := range *l {
		fmt.Fprint(b, el.String())
		if i < len(*l)-1 {
			fmt.Fprint(b, token.Comma, " ")
		}
	}
	fmt.Fprint(b, token.RightBrack)
	return b.String()
}

func (l *loxList) Repr() string {
	return l.String()
}

func (l *loxList) Type() loxType {
	return loxTypeList
}

func (l *loxList) Equals(other loxObject) bool {
	otherList, ok := other.(*loxList)
	if !ok {
		return false
	}
	return slices.EqualFunc(*l, *otherList, func(x, y loxObject) bool {
		return x.Equals(y)
	})
}

func (l *loxList) BinaryOp(op token.Token, right loxObject) loxObject {
rightSwitch:
	switch right := right.(type) {
	case *loxList:
		switch op.Type {
		case token.Plus:
			lCopy := slices.Clone(*l)
			result := append(lCopy, *right...)
			return &result
		default:
			break rightSwitch
		}
	case loxNumber:
		switch op.Type {
		case token.Asterisk:
			return numberTimesList(right, op, l)
		default:
			break rightSwitch
		}
	default:
		break rightSwitch
	}
	panic(newInvalidBinaryOpError(op, l, right))
}

func (l *loxList) Index(index loxObject, node ast.Node) loxObject {
	indexInt := l.indexInt(index, node)
	return (*l)[indexInt]
}

func (l *loxList) SetIndex(index loxObject, node ast.Node, value loxObject) {
	indexInt := l.indexInt(index, node)
	(*l)[indexInt] = value
}

func (l *loxList) indexInt(index loxObject, node ast.Node) int {
	indexNumber, ok := index.(loxNumber)
	if !ok {
		panic(loxerr.Newf(node, loxerr.Fatal, "index (%s) must be a non-negative integer", index.Repr()))
	}
	if math.Floor(float64(indexNumber)) != float64(indexNumber) {
		panic(loxerr.Newf(node, loxerr.Fatal, "index (%s) must be a non-negative integer", indexNumber))
	}
	indexInt := int(indexNumber)
	if indexInt < 0 {
		panic(loxerr.Newf(node, loxerr.Fatal, "index (%s) must not be negative", indexNumber))
	}
	if indexInt >= len(*l) {
		panic(loxerr.Newf(node, loxerr.Fatal, "index %d out of bounds for list of length %v", indexInt, len(*l)))
	}
	return indexInt
}

func (l *loxList) Property(_ *Interpreter, name *ast.Ident) loxObject {
	switch name.String() {
	case "push":
		return newBuiltInLoxMethod("list.push", []string{"value"}, func(args []loxObject) loxObject {
			*l = append(*l, args[0])
			return loxNil{}
		})
	case "pop":
		return newBuiltInLoxMethod("list.pop", []string{}, func([]loxObject) loxObject {
			if len(*l) == 0 {
				return errorMsg(fmt.Sprintf("pop from empty %m", loxTypeList))
			}
			value := (*l)[len(*l)-1]
			*l = (*l)[:len(*l)-1]
			return value
		})
	case "length":
		return loxNumber(len(*l))
	}
	panic(loxerr.Newf(name, loxerr.Fatal, "%m object has no property %m", loxTypeList, name))
}

// errorMsg is a special object which can be returned a callable. It will be caught by the interpreter and converted
// into a runtime error.
type errorMsg string

var _ loxObject = errorMsg("")

func (errorMsg) String() string {
	panic("errorMsg is not a real loxObject")
}

func (errorMsg) Repr() string {
	panic("errorMsg is not a real loxObject")
}

func (errorMsg) Type() loxType {
	panic("errorMsg is not a real loxObject")
}

func (errorMsg) Equals(loxObject) bool {
	panic("errorMsg is not a real loxObject")
}
