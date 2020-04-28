// Copyright (c) 2018-2020, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"jsouthworth.net/go/dyn"
	"jsouthworth.net/go/try"
)

// ValueNew turns a native go value into an RFC7951 Value
// as long as the type can be represented in RFC7951 encoding.
// ValueNew will panic if the value is not an RFC7951 compatible type.
func ValueNew(data interface{}) *Value {
	return valueNew(data)
}

func valueNew(data interface{}) *Value {
	// TODO: Arbitray slices, structs, and map[string]T using reflection
	// Invalid types would be maps[kT]vT where kT is not a string
	// channels.
	if data == nil {
		return &Value{data: nil}
	}
	switch d := data.(type) {
	case *Value:
		return d
	case *Object, *Array, *InstanceID, empty:
	case uint, uint8, uint16, uint32:
		data = convertToUint32(d)
	case uint64:
		// uint64 is marshalled differently than uint32
		// so we need to retain its type.
	case int, int8, int16, int32:
		// Make sure that the unmarshaller and values
		// created this way always yeild the same type
		// otherwise equality won't work. When users
		// unpack the value the proper conversion will
		// be made to the requested type, so storing
		// a positive int32 as a uint32 internally
		// is OK. Since values are immutable the next
		// insertion will be of a different perhaps
		// different internal type.
		data = inferInt32Type(convertToInt32(d))
	case int64:
		// int64 is marshalled differently than int32
		// so we need to retain its type. As is the case
		// for 32 bit integers we need to discover what
		// type to store the value as to ensure consistency
		// with the unmarshal code.
		data = inferInt64Type(d)
	case float32:
		data = float64(d)
	case float64:
	case bool:
	case string:
	case map[string]interface{}:
		data = ObjectFrom(d)
	case []interface{}:
		if len(d) == 1 && d[0] == nil {
			return _empty
		}
		data = ArrayFrom(d)
	default:
		panic(errors.New("cannot create value, invalid type"))
	}
	return &Value{
		data: data,
	}
}

// Value is an RFC7951 value. Values may be *Object, *Array, *InstanceID,
// int32, int64, uint32, uint64, float64, string, bool, Empty or nil.
// All (u)integer types less than 32 are up-converted to a 32bit type when
// creating a value.
type Value struct {
	data interface{}
}

// String is a type that allows differentiation of functions that require
// a go string or an RFC7951 compatible string. It can be used with perform
// to unpack the value correctly depending on the desired semantics.
type String string

var valType = reflect.TypeOf((*Value)(nil))
var interfaceType = reflect.TypeOf((*interface{})(nil)).Elem()
var stringType = reflect.TypeOf(String(""))

// Perform allows one to match the type of the Value with a behavior
// to perform on that type without resulting to the assertion
// operations. Think of this as the switch v.(type) { ... } analogue for
// RFC7951 types. It takes a list of func(v vT) oT functions and applies
// the first match to the value.
//
// If vT above is *Value, String, or interface{} it matches all value
// types. If it is String then RFC7951String is called on the value first. If
// the value is a numeric type and the numeric type is convertable to vT
// then that is considered a match and the conversion is applied first,
// this is not go's standard ConvertibleTo however, only uint32 <-> int32
// and uint64 <-> int64 are supported and only if the values fit.
func (val *Value) Perform(fns ...interface{}) interface{} {
	if val == nil {
		return nil
	}
	vty := reflect.TypeOf(val.data)
	var action interface{}
	arg := val.data
	for _, fn := range fns {
		if action != nil {
			break
		}
		fnty := reflect.TypeOf(fn)
		if fnty.NumIn() != 1 {
			continue
		}
		inputType := fnty.In(0)
		switch {
		case vty == nil:
			if inputType == interfaceType {
				action = fn
			}
		case inputType == valType:
			arg = val
			action = fn
		case inputType == stringType:
			arg = String(val.RFC7951String())
			action = fn
		case vty.AssignableTo(inputType):
			action = fn
		case canConvertNumeric(vty, inputType, arg):
			// Schema less parsing means we don't really know
			// the right numeric type, we use uint32 for all
			// positive numbers but they may actually be int32.
			// Let the user request an int32 if the number fits.
			arg = convertNumeric(arg, inputType)
			action = fn
		}
	}
	if action == nil {
		return nil
	}
	return dyn.Apply(action, arg)
}

func canConvertNumeric(from, to reflect.Type, v interface{}) bool {
	// This is a specific subset of what (reflect.Value).Convert allows
	// we need to be more strict because 32 and 64 bit numbers are treated
	// very differently but we may not know the exact type for positive
	// numbers we receive so we need to allow some automatic conversions.
	if from == to {
		return true
	}
	switch from {
	case int32Type:
		return to == uint32Type && v.(int32) >= 0
	case uint32Type:
		return to == int32Type && v.(uint32) <= ((1<<31)-1)
	case int64Type:
		return to == uint64Type && v.(int64) >= 0
	case uint64Type:
		return to == int64Type && v.(uint64) <= (1<<63)-1
	}
	return false
}

func convertNumeric(from interface{}, to reflect.Type) interface{} {
	return reflect.ValueOf(from).
		Convert(to).
		Interface()
}

// ToTree returns a *Tree if the value is an Object and panics otherwise.
func (val *Value) ToTree() *Tree {
	return val.Perform(func(o *Object) *Tree {
		return TreeFromObject(o)
	}, func(v *Value) *Tree {
		return TreeFromValue(v)
	}).(*Tree)
}

// AsObject returns an *Object if the value is an Object and panics otherwise.
func (val *Value) AsObject() *Object {
	return val.data.(*Object)
}

// IsObject returns if the data stored in the value is an Object.
func (val *Value) IsObject() bool {
	_, isObject := val.data.(*Object)
	return isObject
}

// ToObject returns an *Object and allows the user to define a
// default. The value (*Object)(nil) is returned if no default is defined
// and the value is not an *Object.
func (val *Value) ToObject(defaultVal ...*Object) *Object {
	o, isObject := val.data.(*Object)
	if isObject {
		return o
	}
	if len(defaultVal) != 0 {
		return defaultVal[0]
	}
	return nil
}

// AsArray returns an *Array if the value is an Array and panics otherwise.
func (val *Value) AsArray() *Array {
	return val.data.(*Array)
}

// IsArray returns if the data stored in the value is an Array.
func (val *Value) IsArray() bool {
	_, isArray := val.data.(*Array)
	return isArray
}

// ToArray returns an *Array and allows the user to define a
// default. The value (*Array)(nil) is returned if no default is defined
// and the value is not an *Array.
func (val *Value) ToArray(defaultVal ...*Array) *Array {
	arr, isArray := val.data.(*Array)
	if isArray {
		return arr
	}
	if len(defaultVal) != 0 {
		return defaultVal[0]
	}
	return nil
}

// AsString returns an string if the value is an String and panics otherwise.
func (val *Value) AsString() string {
	return val.data.(string)
}

// IsString returns if the data stored in the value is an String.
func (val *Value) IsString() bool {
	_, isString := val.data.(string)
	return isString
}

// ToString returns an string and allows the user to define a
// default. The value "" is returned if no default is defined
// and the value is not an string.
func (val *Value) ToString(defaultVal ...string) string {
	arr, isString := val.data.(string)
	if isString {
		return arr
	}
	if len(defaultVal) != 0 {
		return defaultVal[0]
	}
	return ""
}

// RFC7951String converts the object to a string that can be encoded in
// RFC7951 format. This may be different than what String returns
// so interface { RFC7951String() string } may be implemented to override
// the behavior here.
func (val *Value) RFC7951String() string {
	if val.data == nil {
		return "null"
	}
	switch d := val.data.(type) {
	case interface {
		RFC7951String() string
	}:
		return d.RFC7951String()
	case uint32:
		return strconv.FormatUint(uint64(d), 10)
	case uint64:
		return strconv.FormatUint(d, 10)
	case int32:
		return strconv.FormatInt(int64(d), 10)
	case int64:
		return strconv.FormatInt(d, 10)
	case float64:
		return strconv.FormatFloat(d, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(d)
	case string:
		str := strconv.Quote(d)
		if len(str) < 2 {
			return str
		}
		// Remove quotes from start/end
		return str[1 : len(str)-1]
	default:
		panic(errors.New("cannot convert value to string, invalid type"))
	}
}

var int32Type = reflect.TypeOf(int32(0))

func convertToInt32(v interface{}) int32 {
	return reflect.ValueOf(v).
		Convert(int32Type).
		Interface().(int32)
}

var uint32Type = reflect.TypeOf(uint32(0))

func convertToUint32(v interface{}) uint32 {
	return reflect.ValueOf(v).
		Convert(uint32Type).
		Interface().(uint32)
}

func inferInt32Type(v int32) interface{} {
	if canConvertNumeric(int32Type, uint32Type, v) {
		return convertToUint32(v)
	}
	return v
}

// AsInt32 returns an int32 if the type is convertable to int32 and panics otherwise.
func (val *Value) AsInt32() int32 {
	return convertToInt32(val.data)
}

// IsInt32 returns if the value is an int32
func (val *Value) IsInt32() bool {
	return canConvertNumeric(reflect.TypeOf(val.data),
		int32Type, val.data)
}

// ToInt32 returns an int32 if the type is convertable to int32 and returns the user supplied default or 0 otherwise.
func (val *Value) ToInt32(defaultVal ...int32) int32 {
	if reflect.TypeOf(val.data).ConvertibleTo(int32Type) {
		return convertToInt32(val.data)
	}
	if len(defaultVal) != 0 {
		return defaultVal[0]
	}
	return 0
}

// AsUint32 returns an uint32 if the type is convertable to uint32 and panics otherwise.
func (val *Value) AsUint32() uint32 {
	return convertToUint32(val.data)
}

// IsUint32 returns if the value is an uint32
func (val *Value) IsUint32() bool {
	return canConvertNumeric(reflect.TypeOf(val.data),
		uint32Type, val.data)
}

// ToUint32 returns an uint32 if the type is convertable to uint32 and returns the user supplied default or 0 otherwise.
func (val *Value) ToUint32(defaultVal ...uint32) uint32 {
	if reflect.TypeOf(val.data).ConvertibleTo(uint32Type) {
		return convertToUint32(val.data)
	}
	if len(defaultVal) != 0 {
		return defaultVal[0]
	}
	return 0
}

var int64Type = reflect.TypeOf(int64(0))

func convertToInt64(v interface{}) int64 {
	return reflect.ValueOf(v).
		Convert(int64Type).
		Interface().(int64)
}

func inferInt64Type(v int64) interface{} {
	if canConvertNumeric(int64Type, uint64Type, v) {
		return convertToUint64(v)
	}
	return v
}

// AsInt64 returns an int64 if the type is convertable to int64 and panics otherwise.
func (val *Value) AsInt64() int64 {
	return convertToInt64(val.data)
}

// IsInt64 returns if the value is an int64
func (val *Value) IsInt64() bool {
	return canConvertNumeric(reflect.TypeOf(val.data),
		int64Type, val.data)
}

// ToInt64 returns an int64 if the type is convertable to int64 and returns the user supplied default or 0 otherwise.
func (val *Value) ToInt64(defaultVal ...int64) int64 {
	if reflect.TypeOf(val.data).ConvertibleTo(int64Type) {
		return convertToInt64(val.data)
	}
	if len(defaultVal) != 0 {
		return defaultVal[0]
	}
	return 0
}

var uint64Type = reflect.TypeOf(uint64(0))

func convertToUint64(v interface{}) uint64 {
	return reflect.ValueOf(v).
		Convert(uint64Type).
		Interface().(uint64)
}

// AsUint64 returns an uint64 if the type is convertable to uint64 and panics otherwise.
func (val *Value) AsUint64() uint64 {
	return convertToUint64(val.data)
}

// IsUint64 returns if the value is an uint64
func (val *Value) IsUint64() bool {
	return canConvertNumeric(reflect.TypeOf(val.data),
		uint64Type, val.data)
}

// ToUint64 returns an uint64 if the type is convertable to uint64 and returns the user supplied default or 0 otherwise.
func (val *Value) ToUint64(defaultVal ...uint64) uint64 {
	if reflect.TypeOf(val.data).ConvertibleTo(uint64Type) {
		return convertToUint64(val.data)
	}
	if len(defaultVal) != 0 {
		return defaultVal[0]
	}
	return 0
}

var float64Type = reflect.TypeOf(float64(0))

func convertToFloat(v interface{}) float64 {
	return reflect.ValueOf(v).
		Convert(float64Type).
		Interface().(float64)
}

// AsFloat returns an float64 if the type is convertable to float64 and panics otherwise.
func (val *Value) AsFloat() float64 {
	return convertToFloat(val.data)
}

// IsFloat returns if the value is an float
func (val *Value) IsFloat() bool {
	_, isFloat := val.data.(float64)
	return isFloat
}

// ToFloat returns an float64 if the type is convertable to float64 and returns the user supplied default or 0 otherwise.
func (val *Value) ToFloat(defaultVal ...float64) float64 {
	if reflect.TypeOf(val.data).ConvertibleTo(float64Type) {
		return convertToFloat(val.data)
	}
	if len(defaultVal) != 0 {
		return defaultVal[0]
	}
	return 0
}

// AsBoolean returns a bool if the value is a bool or if the value is Empty it returns true.
func (val *Value) AsBoolean() bool {
	if val.IsEmpty() {
		return true
	}
	return val.data.(bool)
}

// IsBoolean returns if the value is an bool
func (val *Value) IsBoolean() bool {
	_, isBoolean := val.data.(bool)
	return isBoolean || val.IsEmpty()
}

// ToBoolean returns an bool if the type is convertable to bool and returns the user supplied default or false otherwise.
func (val *Value) ToBoolean(defaultVal ...bool) bool {
	if val.IsEmpty() {
		return true
	}
	b, isBool := val.data.(bool)
	if isBool {
		return b
	}
	if len(defaultVal) != 0 {
		return defaultVal[0]
	}
	return false
}

// ToInterface returns the held data directly as a native interface.
// Caution should be used as the integer types may not be the same as
// the type that was passed into the value due to the way they are
// stored internally. For instance all positive integer values are stored
// as uint32 value and the same goes for int64 and uint64.
func (val *Value) ToInterface() interface{} {
	return val.data
}

// ToData returns the held data as a value that can easily be used with
// standard library packages such as text/template.
func (val *Value) ToData() interface{} {
	switch v := val.data.(type) {
	case interface{ toData() interface{} }:
		return v.toData()
	default:
		return val.data
	}
}

// AsInstanceID returns an instance-identifier if the type is string
// an attempt to parse the instance-identifier will be made.
func (val *Value) AsInstanceID() *InstanceID {
	switch v := val.data.(type) {
	case *InstanceID:
		return v
	case string:
		return InstanceIDNew(v)
	default:
		return v.(*InstanceID) //causes a failure
	}
}

// IsInstanceID returns whether the value is an instance-identifier.
func (val *Value) IsInstanceID() bool {
	switch v := val.data.(type) {
	case *InstanceID:
		return true
	case string:
		_, err := try.Apply(InstanceIDNew, v)
		return err == nil
	default:
		return false
	}
}

// ToInstanceID returns an *InstanceID and allows the user to define a
// default. The value (*InstanceID)(nil) is returned if no default is defined
// and the value is not an *InstanceID.
func (val *Value) ToInstanceID(defaultVal ...*InstanceID) *InstanceID {
	switch v := val.data.(type) {
	case *InstanceID:
		return v
	case string:
		o, err := try.Apply(InstanceIDNew, v)
		if err == nil {
			return o.(*InstanceID)
		}
	}
	if len(defaultVal) != 0 {
		return defaultVal[0]
	}
	return nil
}

// ToNative converts a value to a go native type. It is not recommended
// that this is used as the integer types may not be what you expect
// we store integers in a specific way to ensure the marshaller works
// consistently and it may be different than the type that was inserted.
func (val *Value) ToNative() interface{} {
	switch val := val.data.(type) {
	case interface {
		toNative() interface{}
	}:
		return val.toNative()
	default:
		return val
	}
}

// IsEmpty returns whether a node is the Empty node or not.
func (val *Value) IsEmpty() bool {
	return equal(val, Empty())
}

// IsNull returns whether the values data is nil.
func (val *Value) IsNull() bool {
	return val.data == nil
}

// Merge will combine the old value with the new value and return the
// result.
func (val *Value) Merge(new *Value) *Value {
	switch val := val.data.(type) {
	case interface {
		merge(*Value) *Value
	}:
		return val.merge(new)
	default:
		return new
	}
}

func (val *Value) diff(new *Value, path *InstanceID) []EditEntry {
	switch v := val.data.(type) {
	case interface {
		diff(*Value, *InstanceID) []EditEntry
	}:
		return v.diff(new, path)
	default:
		// Leaf values
		if equal(val, new) {
			return nil
		}
		return []EditEntry{
			{Action: EditAssoc, Path: path, Value: new},
		}
	}
}

// Equal provides an implementation of Equality for Value types.
func (val *Value) Equal(other interface{}) bool {
	if other == nil {
		return val == nil
	}
	ov, isValue := other.(*Value)
	if !isValue {
		return false
	}
	return (val == nil && ov == nil) ||
		equal(val.data, ov.data)
}

// Compare provides an implementation of Comparison for Value types.
func (val *Value) Compare(other interface{}) int {
	return dyn.Compare(val.data, other.(*Value).data)
}

func (val *Value) equal(other *Value) bool {
	return val.data == other.data
}

// String returns a go string representation of the Value.
func (val *Value) String() string {
	return fmt.Sprintf("%v", val.data)
}

func (val *Value) belongsTo(orig *Value, moduleName string) *Value {
	switch v := val.data.(type) {
	case interface {
		belongsTo(*Value, string) *Value
	}:
		return v.belongsTo(val, moduleName)
	default:
		return val
	}
}

func (val *Value) marshalRFC7951(buf *bytes.Buffer, module string) error {
	switch v := val.data.(type) {
	case interface {
		marshalRFC7951(*bytes.Buffer, string) error
	}:
		return v.marshalRFC7951(buf, module)
	case uint64, int64, float32, float64, string:
		buf.WriteByte('"')
		buf.WriteString(val.RFC7951String())
		buf.WriteByte('"')
	default:
		buf.WriteString(val.RFC7951String())
	}
	return nil
}

// MarshalRFC7951 returns the value encoded in an RFC7951 compatible way.
func (val *Value) MarshalRFC7951() ([]byte, error) {
	var buf bytes.Buffer
	err := val.marshalRFC7951(&buf, "")
	return buf.Bytes(), err
}

// UnmarshalRFC7951 extracts a value from an rfc7951 encoded value.
func (val *Value) UnmarshalRFC7951(msg []byte) error {
	strs := stringInternerNew()
	vals := valueInternerNew()
	return val.unmarshalRFC7951(msg, "", strs, vals)
}

func (val *Value) unmarshalRFC7951(
	msg []byte, module string,
	strs *stringInterner,
	vals *valueInterner,
) error {
	if len(msg) == 0 {
		return nil
	}
	switch c := msg[0]; c {
	case '{':
		obj := objectNew()
		err := obj.unmarshalRFC7951(msg, module, strs, vals)
		if err != nil {
			return err
		}
		val.data = obj
	case '[':
		arr := arrayNew()
		err := arr.unmarshalRFC7951(msg, module, strs, vals)
		if err != nil {
			return err
		}
		if arr.Length() == 1 && equal(arr.At(0), ValueNew(nil)) {
			val.data = _empty.data
			return nil
		}
		val.data = arr
	case 'n':
		val.data = nil
	case 't', 'f':
		val.data = c == 't'
	case '"':
		// Quoted values may be strings, int64, uint64, or
		// floating point numbers in RFC7951 encoding.  Attempt
		// to decode into the correct type without knowing the
		// actual schema. Callers may use the As* assertions to
		// access as the actual data type.
		item, err := strconv.Unquote(string(msg))
		if err != nil {
			return err
		}
		item = strs.Intern(item)
		if len(item) == 0 {
			val.data = item
			return nil
		}
		c := item[0]
		switch {
		case c == '-' && len(item) >= 2:
			n := item[1]
			if n < '0' || n > '9' {
				val.data = item
				return nil
			}
			if strings.Contains(item, ".") {
				f, err := strconv.ParseFloat(item, 64)
				if err != nil {
					//it wasn't a float, use the string
					val.data = item
					return nil
				}
				val.data = f
			}
			i, err := strconv.ParseInt(item, 10, 64)
			if err != nil {
				//it wasn't an int, use the string
				val.data = item
				return nil
			}
			val.data = i
		case c == '+' && len(item) >= 2:
			n := item[1]
			if n < '0' || n > '9' {
				val.data = item
				return nil
			}
			if strings.Contains(item, ".") {
				f, err := strconv.ParseFloat(item[1:], 64)
				if err != nil {
					//it wasn't a float, use the string
					val.data = item
					return nil
				}
				val.data = f
			}
			i, err := strconv.ParseUint(item[1:], 10, 64)
			if err != nil {
				//it wasn't an int, use the string
				val.data = item
				return nil
			}
			val.data = i
		case c >= '0' && c <= '9':
			if strings.Contains(item, ".") {
				f, err := strconv.ParseFloat(item, 64)
				if err != nil {
					//it wasn't a float, use the string
					val.data = item
					return nil
				}
				val.data = f
			}
			i, err := strconv.ParseUint(item, 10, 64)
			if err != nil {
				//it wasn't an int, use the string
				val.data = item
				return nil
			}
			val.data = i
		default:
			val.data = item
		}
	case '-':
		i, err := strconv.ParseInt(string(msg), 10, 32)
		if err != nil {
			return err
		}
		val.data = int32(i)
	default:
		i, err := strconv.ParseUint(string(msg), 10, 32)
		if err != nil {
			return err
		}
		val.data = uint32(i)
	}
	return nil
}

var _empty = &Value{data: empty{}}

// Empty returns the constant empty value
func Empty() *Value {
	return _empty
}

type empty struct{}

func (empty) toNative() interface{} {
	return []interface{}{nil}
}

func (empty) RFC7951String() string {
	return "[null]"
}

func equal(v1, v2 interface{}) bool {
	return dyn.Equal(v1, v2)
}
