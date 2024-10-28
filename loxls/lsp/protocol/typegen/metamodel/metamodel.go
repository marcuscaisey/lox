// Package metamodel contains the types that make up the LSP meta model and a function to load the meta model from
// Microsoft's website.
package metamodel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
)

// AndType represents an `and`type (e.g. TextDocumentParams & WorkDoneProgressParams`).
type AndType struct {
	Items []*Type `json:"items"`
	Kind  string  `json:"kind"`
}

// ArrayType represents an array type (e.g. `TextDocument[]`).
type ArrayType struct {
	Element *Type  `json:"element"`
	Kind    string `json:"kind"`
}

// BaseType represents a base type like `string` or `DocumentUri`.
type BaseType struct {
	Kind string    `json:"kind"`
	Name BaseTypes `json:"name"`
}

// BaseTypes represents the possible base types.
type BaseTypes string

// Possible [BaseTypes] values.
const (
	BaseTypesURI         BaseTypes = "URI"
	BaseTypesDocumentURI BaseTypes = "DocumentUri"
	BaseTypesInteger     BaseTypes = "integer"
	BaseTypesUinteger    BaseTypes = "uinteger"
	BaseTypesDecimal     BaseTypes = "decimal"
	BaseTypesRegExp      BaseTypes = "RegExp"
	BaseTypesString      BaseTypes = "string"
	BaseTypesBoolean     BaseTypes = "boolean"
	BaseTypesNull        BaseTypes = "null"
)

// BooleanLiteralType represents a boolean literal type (e.g. `kind: true`).
type BooleanLiteralType struct {
	Kind  string `json:"kind"`
	Value bool   `json:"value"`
}

// Enumeration defines an Enumeration.
type Enumeration struct {
	// Whether the enumeration is deprecated or not. If deprecated the property contains the deprecation message.
	Deprecated *string `json:"deprecated,omitempty"`
	// An optional documentation.
	Documentation *string `json:"documentation,omitempty"`
	// The name of the enumeration.
	Name string `json:"name"`
	// Whether this is a proposed enumeration. If omitted, the enumeration is final.
	Proposed *bool `json:"proposed,omitempty"`
	// Since when (release number) this enumeration is available. Is undefined if not known.
	Since *string `json:"since,omitempty"`
	// Whether the enumeration supports custom values (e.g. values which are not part of the set defined in `values`).
	// If omitted no custom values are supported.
	SupportsCustomValues *bool `json:"supportsCustomValues,omitempty"`
	// The type of the elements.
	Type EnumerationType `json:"type"`
	// The enum values.
	Values []*EnumerationEntry `json:"values"`
}

// EnumerationEntry defines an enumeration entry.
type EnumerationEntry struct {
	// Whether the enum entry is deprecated or not. If deprecated the property contains the deprecation message.
	Deprecated *string `json:"deprecated,omitempty"`
	// An optional documentation.
	Documentation *string `json:"documentation,omitempty"`
	// The name of the enum item.
	Name string `json:"name"`
	// Whether this is a proposed enumeration entry. If omitted, the enumeration entry is final.
	Proposed *bool `json:"proposed,omitempty"`
	// Since when (release number) this enumeration entry is available. Is undefined if not known.
	Since *string `json:"since,omitempty"`
	// The value.
	Value IntOrString `json:"value"`
}

// EnumerationType represents the type of an enumeration.
type EnumerationType struct {
	Kind string              `json:"kind"`
	Name EnumerationTypeName `json:"name"`
}

// EnumerationTypeName represents the possible enumeration type names.
type EnumerationTypeName string

// Possible [EnumerationTypeName] values.
const (
	EnumerationTypeNameString   EnumerationTypeName = "string"
	EnumerationTypeNameInteger  EnumerationTypeName = "integer"
	EnumerationTypeNameUinteger EnumerationTypeName = "uinteger"
)

// IntegerLiteralType represents an integer literal type (e.g. `kind: 1`).
type IntegerLiteralType struct {
	Kind  string  `json:"kind"`
	Value float64 `json:"value"`
}

// MapKeyType represents a type that can be used as a key in a map type. If a reference type is used then the type must
// either resolve to a `string` or `integer` type. (e.g. `type ChangeAnnotationIdentifier === string`).
type MapKeyType struct {
	Value MapKeyTypeValue
}

// MapKeyTypeValue is either of the following types:
//   - [BaseMapKeyType]
//   - [ReferenceType]
type MapKeyTypeValue interface {
	isMapKeyTypeValue()
}

func (BaseMapKeyType) isMapKeyTypeValue() {}
func (ReferenceType) isMapKeyTypeValue()  {}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (m *MapKeyType) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	var value struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	switch value.Kind {
	case "base":
		var baseMapKeyTypeValue BaseMapKeyType
		if err := json.Unmarshal(data, &baseMapKeyTypeValue); err != nil {
			return err
		}
		m.Value = baseMapKeyTypeValue
		return nil
	case "reference":
		var referenceTypeValue ReferenceType
		if err := json.Unmarshal(data, &referenceTypeValue); err != nil {
			return err
		}
		m.Value = referenceTypeValue
		return nil
	}
	return &json.UnmarshalTypeError{
		Value: string(data),
		Type:  reflect.TypeFor[*Type](),
	}
}

// MarshalJSON implements the [json.Marshaler] interface.
func (m MapKeyType) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Value)
}

// BaseMapKeyType represents a base map key type.
type BaseMapKeyType struct {
	Kind string             `json:"kind"`
	Name BaseMapKeyTypeName `json:"name"`
}

// BaseMapKeyTypeName represents the possible base map key type names.
type BaseMapKeyTypeName string

// Possible [BaseMapKeyTypeName] values.
const (
	BaseMapKeyTypeNameURI         BaseMapKeyTypeName = "URI"
	BaseMapKeyTypeNameDocumentURI BaseMapKeyTypeName = "DocumentUri"
	BaseMapKeyTypeNameString      BaseMapKeyTypeName = "string"
	BaseMapKeyTypeNameInteger     BaseMapKeyTypeName = "integer"
)

// MapType represents a JSON object map (e.g. `interface Map<K extends string | integer, V> { [key: K] => V; }`).
type MapType struct {
	Key   MapKeyType `json:"key"`
	Kind  string     `json:"kind"`
	Value *Type      `json:"value"`
}

// MessageDirection indicates in which direction a message is sent in the protocol.
type MessageDirection string

// Possible [MessageDirection] values.
const (
	MessageDirectionClientToServer MessageDirection = "clientToServer"
	MessageDirectionServerToClient MessageDirection = "serverToClient"
	MessageDirectionBoth           MessageDirection = "both"
)

// MetaData contains additional meta data.
type MetaData struct {
	// The protocol version.
	Version string `json:"version"`
}

// MetaModel is the actual meta model.
type MetaModel struct {
	// The enumerations.
	Enumerations []*Enumeration `json:"enumerations"`
	// Additional meta data.
	MetaData MetaData `json:"metaData"`
	// The notifications.
	Notifications []*Notification `json:"notifications"`
	// The requests.
	Requests []*Request `json:"requests"`
	// The structures.
	Structures []*Structure `json:"structures"`
	// The type aliases.
	TypeAliases []*TypeAlias `json:"typeAliases"`
}

// MethodTypes returns the types associated with the given methods.
// For a request, the types of the params, result, and error data are returned.
// For a notification, the type of the params is returned.
func (m *MetaModel) MethodTypes(methods []string) ([]*Type, error) {
	var types []*Type
	invalidMethods := slices.Clone(methods)

	for _, req := range m.Requests {
		i := slices.Index(invalidMethods, req.Method)
		if i == -1 {
			continue
		}
		invalidMethods = slices.Delete(invalidMethods, i, i+1)

		types = append(types, req.Params.Flatten()...)
		types = append(types, req.Result)
		if req.ErrorData != nil {
			types = append(types, req.ErrorData)
		}
	}

	for _, notif := range m.Notifications {
		i := slices.Index(invalidMethods, notif.Method)
		if i == -1 {
			continue
		}
		invalidMethods = slices.Delete(invalidMethods, i, i+1)

		types = append(types, notif.Params.Flatten()...)
	}

	if len(invalidMethods) > 0 {
		return nil, fmt.Errorf("the following methods are invalid: %s", strings.Join(invalidMethods, ", "))
	}

	return types, nil
}

// Structure returns the [Structure] with the given name and whether it exists.
func (m *MetaModel) Structure(name string) (value *Structure, ok bool) {
	for _, structure := range m.Structures {
		if name == structure.Name {
			return structure, true
		}
	}
	return nil, false
}

// TypeAlias returns the [TypeAlias] with the given name and whether it exists.
func (m *MetaModel) TypeAlias(name string) (value *TypeAlias, ok bool) {
	for _, typeAlias := range m.TypeAliases {
		if name == typeAlias.Name {
			return typeAlias, true
		}
	}
	return nil, false
}

// Enumeration returns the [Enumeration] with the given name and whether it exists.
func (m *MetaModel) Enumeration(name string) (value *Enumeration, ok bool) {
	for _, enum := range m.Enumerations {
		if name == enum.Name {
			return enum, true
		}
	}
	return nil, false
}

// Notification represents a LSP Notification
type Notification struct {
	// Whether the notification is deprecated or not. If deprecated the property contains the deprecation message.
	Deprecated *string `json:"deprecated,omitempty"`
	// An optional documentation;
	Documentation *string `json:"documentation,omitempty"`
	// The direction in which this notification is sent in the protocol.
	MessageDirection MessageDirection `json:"messageDirection"`
	// The request's method name.
	Method string `json:"method"`
	// The parameter type(s) if any.
	Params *TypeOrTypeSlice `json:"params,omitempty"`
	// Whether this is a proposed notification. If omitted the notification is final.
	Proposed *bool `json:"proposed,omitempty"`
	// Optional a dynamic registration method if it different from the request's method.
	RegistrationMethod *string `json:"registrationMethod,omitempty"`
	// Optional registration options if the notification supports dynamic registration.
	RegistrationOptions *Type `json:"registrationOptions,omitempty"`
	// Since when (release number) this notification is available. Is undefined if not known.
	Since *string `json:"since,omitempty"`
}

// TypeOrTypeSlice contains either of the following types:
//   - [*Type]
//   - [TypeSlice]
type TypeOrTypeSlice struct {
	Value TypeOrTypeSliceValue
}

// Flatten returns a slice of the types.
func (t *TypeOrTypeSlice) Flatten() []*Type {
	if t == nil {
		return nil
	}
	switch value := t.Value.(type) {
	case *Type:
		return []*Type{value}
	case TypeSlice:
		return value
	default:
		panic("unreachable")
	}
}

// TypeOrTypeSliceValue is either of the following types:
//   - [*Type]
//   - [TypeSlice]
type TypeOrTypeSliceValue interface {
	isParamsValue()
}

func (*Type) isParamsValue()     {}
func (TypeSlice) isParamsValue() {}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (t *TypeOrTypeSlice) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	var typeValue *Type
	if err := json.Unmarshal(b, &typeValue); err == nil {
		t.Value = typeValue
		return nil
	}
	var arrayValue TypeSlice
	if err := json.Unmarshal(b, &arrayValue); err == nil {
		t.Value = arrayValue
		return nil
	}
	return &json.UnmarshalTypeError{
		Value: string(b),
		Type:  reflect.TypeFor[*TypeOrTypeSlice](),
	}
}

// MarshalJSON implements the [json.Marshaler] interface.
func (t *TypeOrTypeSlice) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Value)
}

// TypeSlice is a slice of [*Type].
type TypeSlice []*Type

// OrType represents an `or` type (e.g. `Location | LocationLink`).
type OrType struct {
	Items []*Type `json:"items"`
	Kind  string  `json:"kind"`
}

// Property represents an object Property.
type Property struct {
	// Whether the property is deprecated or not. If deprecated the property contains the deprecation message.
	Deprecated *string `json:"deprecated,omitempty"`
	// An optional documentation.
	Documentation *string `json:"documentation,omitempty"`
	// The property name;
	Name string `json:"name"`
	// Whether the property is optional. If omitted, the property is mandatory.
	Optional *bool `json:"optional,omitempty"`
	// Whether this is a proposed property. If omitted, the structure is final.
	Proposed *bool `json:"proposed,omitempty"`
	// Since when (release number) this property is available. Is undefined if not known.
	Since *string `json:"since,omitempty"`
	// The type of the property
	Type *Type `json:"type"`
}

// ReferenceType repesents a reference to another type (e.g. `TextDocument`). This is either a `Structure`, a
// `Enumeration` or a `TypeAlias` in the same meta model.
type ReferenceType struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}

// Request represents a LSP Request
type Request struct {
	// Whether the request is deprecated or not. If deprecated the property contains the deprecation message.
	Deprecated *string `json:"deprecated,omitempty"`
	// An optional documentation;
	Documentation *string `json:"documentation,omitempty"`
	// An optional error data type.
	ErrorData *Type `json:"errorData,omitempty"`
	// The direction in which this request is sent in the protocol.
	MessageDirection MessageDirection `json:"messageDirection"`
	// The request's method name.
	Method string `json:"method"`
	// The parameter type(s) if any.
	Params *TypeOrTypeSlice `json:"params,omitempty"`
	// Optional partial result type if the request supports partial result reporting.
	PartialResult *Type `json:"partialResult,omitempty"`
	// Whether this is a proposed feature. If omitted the feature is final.
	Proposed *bool `json:"proposed,omitempty"`
	// Optional a dynamic registration method if it different from the request's method.
	RegistrationMethod *string `json:"registrationMethod,omitempty"`
	// Optional registration options if the request supports dynamic registration.
	RegistrationOptions *Type `json:"registrationOptions,omitempty"`
	// The result type.
	Result *Type `json:"result"`
	// Since when (release number) this request is available. Is undefined if not known.
	Since *string `json:"since,omitempty"`
}

// StringLiteralType represents a string literal type (e.g. `kind: 'rename'`).
type StringLiteralType struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

// Structure defines the Structure of an object literal.
type Structure struct {
	// Whether the structure is deprecated or not. If deprecated the property contains the deprecation message.
	Deprecated *string `json:"deprecated,omitempty"`
	// An optional documentation;
	Documentation *string `json:"documentation,omitempty"`
	// Structures extended from. This structures form a polymorphic type hierarchy.
	Extends []*Type `json:"extends,omitempty"`
	// Structures to mix in. The properties of these structures are `copied` into this structure. Mixins don't form a
	// polymorphic type hierarchy in LSP.
	Mixins []*Type `json:"mixins,omitempty"`
	// The name of the structure.
	Name string `json:"name"`
	// The properties.
	Properties []*Property `json:"properties"`
	// Whether this is a proposed structure. If omitted, the structure is final.
	Proposed *bool `json:"proposed,omitempty"`
	// Since when (release number) this structure is available. Is undefined if not known.
	Since *string `json:"since,omitempty"`
}

// StructureLiteral defines an unnamed structure of an object literal.
type StructureLiteral struct {
	// Whether the literal is deprecated or not. If deprecated the property contains the deprecation message.
	Deprecated *string `json:"deprecated,omitempty"`
	// An optional documentation.
	Documentation *string `json:"documentation,omitempty"`
	// The properties.
	Properties []*Property `json:"properties"`
	// Whether this is a proposed structure. If omitted, the structure is final.
	Proposed *bool `json:"proposed,omitempty"`
	// Since when (release number) this structure is available. Is undefined if not known.
	Since *string `json:"since,omitempty"`
}

// StructureLiteralType represents a literal structure (e.g. `property: { start: uinteger; end: uinteger; }`).
type StructureLiteralType struct {
	Kind  string           `json:"kind"`
	Value StructureLiteral `json:"value"`
}

// TupleType represents a `tuple` type (e.g. `[integer, integer]`).
type TupleType struct {
	Items []*Type `json:"items"`
	Kind  string  `json:"kind"`
}

// Type represents a type.
type Type struct {
	Value TypeValue
}

// TypeValue is either of the following types:
//   - [AndType]
//   - [ArrayType]
//   - [BaseType]
//   - [BooleanLiteralType]
//   - [IntegerLiteralType]
//   - [MapType]
//   - [OrType]
//   - [ReferenceType]
//   - [StringLiteralType]
//   - [StructureLiteralType]
//   - [TupleType]
type TypeValue interface {
	isTypeValue()
}

func (AndType) isTypeValue()              {}
func (ArrayType) isTypeValue()            {}
func (BaseType) isTypeValue()             {}
func (BooleanLiteralType) isTypeValue()   {}
func (IntegerLiteralType) isTypeValue()   {}
func (MapType) isTypeValue()              {}
func (OrType) isTypeValue()               {}
func (ReferenceType) isTypeValue()        {}
func (StringLiteralType) isTypeValue()    {}
func (StructureLiteralType) isTypeValue() {}
func (TupleType) isTypeValue()            {}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (t *Type) UnmarshalJSON(data []byte) error {
	if bytes.Equal(data, []byte("null")) {
		return nil
	}
	var value struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	switch value.Kind {
	case "base":
		var baseTypeValue BaseType
		if err := json.Unmarshal(data, &baseTypeValue); err != nil {
			return err
		}
		t.Value = baseTypeValue
		return nil
	case "reference":
		var referenceTypeValue ReferenceType
		if err := json.Unmarshal(data, &referenceTypeValue); err != nil {
			return err
		}
		t.Value = referenceTypeValue
		return nil
	case "array":
		var arrayTypeValue ArrayType
		if err := json.Unmarshal(data, &arrayTypeValue); err != nil {
			return err
		}
		t.Value = arrayTypeValue
		return nil
	case "map":
		var mapTypeValue MapType
		if err := json.Unmarshal(data, &mapTypeValue); err != nil {
			return err
		}
		t.Value = mapTypeValue
		return nil
	case "and":
		var andTypeValue AndType
		if err := json.Unmarshal(data, &andTypeValue); err != nil {
			return err
		}
		t.Value = andTypeValue
		return nil
	case "or":
		var orTypeValue OrType
		if err := json.Unmarshal(data, &orTypeValue); err != nil {
			return err
		}
		t.Value = orTypeValue
		return nil
	case "tuple":
		var tupleTypeValue TupleType
		if err := json.Unmarshal(data, &tupleTypeValue); err != nil {
			return err
		}
		t.Value = tupleTypeValue
		return nil
	case "literal":
		var structureLiteralTypeValue StructureLiteralType
		if err := json.Unmarshal(data, &structureLiteralTypeValue); err != nil {
			return err
		}
		t.Value = structureLiteralTypeValue
		return nil
	case "stringLiteral":
		var stringLiteralTypeValue StringLiteralType
		if err := json.Unmarshal(data, &stringLiteralTypeValue); err != nil {
			return err
		}
		t.Value = stringLiteralTypeValue
		return nil
	case "integerLiteral":
		var integerLiteralTypeValue IntegerLiteralType
		if err := json.Unmarshal(data, &integerLiteralTypeValue); err != nil {
			return err
		}
		t.Value = integerLiteralTypeValue
		return nil
	case "booleanLiteral":
		var booleanLiteralTypeValue BooleanLiteralType
		if err := json.Unmarshal(data, &booleanLiteralTypeValue); err != nil {
			return err
		}
		t.Value = booleanLiteralTypeValue
		return nil
	}
	return &json.UnmarshalTypeError{
		Value: string(data),
		Type:  reflect.TypeFor[*Type](),
	}
}

// MarshalJSON implements the [json.Marshaler] interface.
func (t *Type) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Value)
}

// TypeAlias defines a type alias. (e.g. `type Definition = Location | LocationLink`)
type TypeAlias struct {
	// Whether the type alias is deprecated or not. If deprecated the property contains the deprecation message.
	Deprecated *string `json:"deprecated,omitempty"`
	// An optional documentation.
	Documentation *string `json:"documentation,omitempty"`
	// The name of the type alias.
	Name string `json:"name"`
	// Whether this is a proposed type alias. If omitted, the type alias is final.
	Proposed *bool `json:"proposed,omitempty"`
	// Since when (release number) this structure is available. Is undefined if not known.
	Since *string `json:"since,omitempty"`
	// The aliased type.
	Type *Type `json:"type"`
}

// TypeKind represents the kind of a type.
type TypeKind string

// Possible [TypeKind] values.
const (
	TypeKindBase           TypeKind = "base"
	TypeKindReference      TypeKind = "reference"
	TypeKindArray          TypeKind = "array"
	TypeKindMap            TypeKind = "map"
	TypeKindAnd            TypeKind = "and"
	TypeKindOr             TypeKind = "or"
	TypeKindTuple          TypeKind = "tuple"
	TypeKindLiteral        TypeKind = "literal"
	TypeKindStringLiteral  TypeKind = "stringLiteral"
	TypeKindIntegerLiteral TypeKind = "integerLiteral"
	TypeKindBooleanLiteral TypeKind = "booleanLiteral"
)

// IntOrString contains either of the following types:
//   - [Int]
//   - [String]
type IntOrString struct {
	Value IntOrStringValue
}

// IntOrStringValue is either of the following types:
//   - [Int]
//   - [String]
type IntOrStringValue interface {
	isIntOrStringValue()
}

func (Int) isIntOrStringValue()    {}
func (String) isIntOrStringValue() {}

// UnmarshalJSON implements the [json.Unmarshaler] interface.
func (i *IntOrString) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}
	var intValue Int
	if err := json.Unmarshal(b, &intValue); err == nil {
		i.Value = intValue
		return nil
	}
	var stringValue String
	if err := json.Unmarshal(b, &stringValue); err == nil {
		i.Value = stringValue
		return nil
	}
	return &json.UnmarshalTypeError{
		Value: string(b),
		Type:  reflect.TypeFor[*IntOrString](),
	}
}

// MarshalJSON implements the [json.Marshaler] interface.
func (i IntOrString) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.Value)
}

// Int wraps the built-in int type.
type Int int

// String wraps the built-in string type.
type String string
