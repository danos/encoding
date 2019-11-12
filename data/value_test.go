// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"os"
	"reflect"
	"testing"
	"text/template"

	"github.com/danos/encoding/rfc7951"
	"jsouthworth.net/go/try"
)

func TestValueNew(t *testing.T) {
	cases := []struct {
		name  string
		rtype reflect.Type
		val   interface{}
	}{
		{"Value", reflect.TypeOf(""), ValueNew("foo")},
		{"Object", reflect.TypeOf((*Object)(nil)), ObjectNew()},
		{"Array", reflect.TypeOf((*Array)(nil)), ArrayNew()},
		{"int8", uint32Type, int8(0)},
		{"int8-neg", int32Type, int16(-1)},
		{"int16", uint32Type, int32(0)},
		{"int16-neg", int32Type, int16(-1)},
		{"int", uint32Type, int(0)},
		{"int-neg", int32Type, int(-1)},
		{"int32", uint32Type, int32(0)},
		{"int32-neg", int32Type, int32(-1)},
		{"uint8", uint32Type, uint8(0)},
		{"uint16", uint32Type, uint32(0)},
		{"uint", uint32Type, uint(0)},
		{"uint32", uint32Type, uint32(0)},
		{"float32", reflect.TypeOf(float64(0)), float32(0)},
		{"float64", reflect.TypeOf(float64(0)), float64(0)},
		{"bool", reflect.TypeOf(false), false},
		{"string", reflect.TypeOf(""), "foo"},
		{"map[string]interface{}", reflect.TypeOf((*Object)(nil)),
			map[string]interface{}{}},
		{"[]interface{}", reflect.TypeOf((*Array)(nil)),
			[]interface{}{}},
		{"[]interface{nil}", reflect.TypeOf(empty{}),
			[]interface{}{nil}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			val := ValueNew(test.val)
			got := reflect.TypeOf(val.data)
			if got != test.rtype {
				t.Fatal("didn't get expected type for value",
					val, got, test.rtype)
			}
		})
	}
}

func TestValuePerform(t *testing.T) {
	cases := []struct {
		name     string
		val      *Value
		fns      []interface{}
		expected interface{}
	}{
		{
			name: "nil",
			val:  ValueNew(nil),
			fns: []interface{}{
				func(v interface{}) interface{} {
					if v == nil {
						return "got it"
					}
					return nil
				},
			},
			expected: "got it",
		},
		{
			name: "Value",
			val:  ValueNew(10),
			fns: []interface{}{
				func(v *Value) *Value {
					return v
				},
			},
			expected: ValueNew(10),
		},
		{
			name: "skip invalid handlers",
			val:  ValueNew(int32(100)),
			fns: []interface{}{
				func(s String, other interface{}) String {
					return s
				},
			},
			expected: nil,
		},
		{
			name: "nil value",
			val:  nil,
			fns: []interface{}{
				func(s String) String {
					return s
				},
			},
			expected: nil,
		},
		// (u)int32 tests
		{
			name: "int32",
			val:  ValueNew(100),
			fns: []interface{}{
				func(i int32) int32 {
					return i
				},
			},
			expected: int32(100),
		},
		{
			name: "uint32->int32",
			val:  ValueNew(uint32(100)),
			fns: []interface{}{
				func(i int32) int32 {
					return i
				},
			},
			expected: int32(100),
		},
		{
			name: "int32->uint32",
			val:  ValueNew(int32(100)),
			fns: []interface{}{
				func(i uint32) uint32 {
					return i
				},
			},
			expected: uint32(100),
		},
		{
			name: "int32->uint32, neg",
			val:  ValueNew(int32(-100)),
			fns: []interface{}{
				func(i uint32) uint32 {
					return i
				},
			},
			expected: nil,
		},
		{
			name: "uint32->int32, big",
			val:  ValueNew(uint32(1 << 31)),
			fns: []interface{}{
				func(i int32) int32 {
					return i
				},
			},
			expected: nil,
		},
		{
			name: "big with 2 handlers",
			val:  ValueNew(uint32(1 << 31)),
			fns: []interface{}{
				func(i int32) int32 {
					return i
				},
				func(i uint32) uint32 {
					return i
				},
			},
			expected: uint32(1 << 31),
		},
		{
			name: "negative with 2 handlers",
			val:  ValueNew(-100),
			fns: []interface{}{
				func(i int32) int32 {
					return i
				},
				func(i uint32) uint32 {
					return i
				},
			},
			expected: int32(-100),
		},
		{
			name: "int32->String",
			val:  ValueNew(int32(100)),
			fns: []interface{}{
				func(s String) String {
					return s
				},
			},
			expected: String("100"),
		},

		// (u)int64 tests
		{
			name: "int64",
			val:  ValueNew(int64(100)),
			fns: []interface{}{
				func(i int64) int64 {
					return i
				},
			},
			expected: int64(100),
		},
		{
			name: "uint64->int64",
			val:  ValueNew(uint64(100)),
			fns: []interface{}{
				func(i int64) int64 {
					return i
				},
			},
			expected: int64(100),
		},
		{
			name: "int64->uint64",
			val:  ValueNew(int64(100)),
			fns: []interface{}{
				func(i uint64) uint64 {
					return i
				},
			},
			expected: uint64(100),
		},
		{
			name: "int64->uint64, neg",
			val:  ValueNew(int64(-100)),
			fns: []interface{}{
				func(i uint64) uint64 {
					return i
				},
			},
			expected: nil,
		},
		{
			name: "uint64->int64, big",
			val:  ValueNew(uint64(1 << 63)),
			fns: []interface{}{
				func(i int64) int64 {
					return i
				},
			},
			expected: nil,
		},
		{
			name: "uint64big with 2 handlers",
			val:  ValueNew(uint64(1 << 63)),
			fns: []interface{}{
				func(i int64) int64 {
					return i
				},
				func(i uint64) uint64 {
					return i
				},
			},
			expected: uint64(1 << 63),
		},
		{
			name: "int64 negative with 2 handlers",
			val:  ValueNew(int64(-100)),
			fns: []interface{}{
				func(i int64) int64 {
					return i
				},
				func(i uint64) uint64 {
					return i
				},
			},
			expected: int64(-100),
		},
		{
			name: "invalid conversion",
			val:  ValueNew("foo"),
			fns: []interface{}{
				func(i int32) int32 {
					return i
				},
				func(i uint64) uint64 {
					return i
				},
			},
			expected: nil,
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := test.val.Perform(test.fns...)
			if !equal(got, test.expected) {
				t.Fatalf("got %T(%v) expected %T(%v)\n",
					got, got, test.expected, test.expected)
			}
		})
	}
}

func TestValueRFC7951String(t *testing.T) {
	cases := []struct {
		name     string
		val      *Value
		expected string
	}{
		{"uint32", ValueNew(10), "10"},
		{"uint64", ValueNew(uint64(10)), "10"},
		{"int32", ValueNew(int32(-1)), "-1"},
		{"int64", ValueNew(int64(-1)), "-1"},
		{"float64", ValueNew(float64(10.1)), "10.1"},
		{"bool", ValueNew(true), "true"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			got := test.val.RFC7951String()
			if !equal(got, test.expected) {
				t.Fatalf("got %T(%v) expected %T(%v)\n",
					got, got, test.expected, test.expected)
			}
		})
	}
}

func TestValueConversions(t *testing.T) {
	// Tree conversion
	t.Run("ToTree", func(t *testing.T) {
		t.Run("Object", func(t *testing.T) {
			v := ValueNew(ObjectWith(PairNew("m:foo", "bar")))
			tree := v.ToTree()
			if !equal(tree.At("/m:foo"), ValueNew("bar")) {
				t.Fatal("didn't get expected result")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			tree := v.ToTree()
			if !equal(tree.At("/rfc7951:data"), v) {
				t.Fatal("didn't get expected result")
			}
		})
	})

	// Object conversion
	t.Run("AsObject", func(t *testing.T) {
		t.Run("Object", func(t *testing.T) {
			v := ValueNew(ObjectWith(PairNew("m:foo", "bar")))
			_ = v.AsObject()
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			_, err := try.Apply(v.AsObject)
			if err == nil {
				t.Fatal("conversion should have failed")
			}
		})
	})
	t.Run("IsObject", func(t *testing.T) {
		t.Run("Object", func(t *testing.T) {
			v := ValueNew(ObjectWith(PairNew("m:foo", "bar")))
			if !v.IsObject() {
				t.Fatal("Value is an object")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			if v.IsObject() {
				t.Fatal("Value is not an object")
			}
		})
	})
	t.Run("ToObject", func(t *testing.T) {
		t.Run("Object", func(t *testing.T) {
			v := ValueNew(ObjectWith(PairNew("m:foo", "bar")))
			o := v.ToObject()
			if !equal(o.At("m:foo"), ValueNew("bar")) {
				t.Fatal("didn't get expected result")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToObject()
			if o != nil {
				t.Fatal("Value should not an object")
			}
		})
		t.Run("Default", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToObject(ObjectNew())
			if o == nil {
				t.Fatal("should have gotten default")
			}
		})
	})

	// Array conversion
	t.Run("AsArray", func(t *testing.T) {
		t.Run("Array", func(t *testing.T) {
			v := ValueNew(ArrayWith("foo", "bar"))
			_ = v.AsArray()
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			_, err := try.Apply(v.AsArray)
			if err == nil {
				t.Fatal("conversion should have failed")
			}
		})
	})
	t.Run("IsArray", func(t *testing.T) {
		t.Run("Array", func(t *testing.T) {
			v := ValueNew(ArrayWith("foo", "bar"))
			if !v.IsArray() {
				t.Fatal("Value is an array")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			if v.IsArray() {
				t.Fatal("Value is not an array")
			}
		})
	})
	t.Run("ToArray", func(t *testing.T) {
		t.Run("Array", func(t *testing.T) {
			v := ValueNew(ArrayWith("foo", "bar"))
			o := v.ToArray()
			if !equal(o.At(1), ValueNew("bar")) {
				t.Fatal("didn't get expected result")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToArray()
			if o != nil {
				t.Fatal("Value should not an array")
			}
		})
		t.Run("Default", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToArray(ArrayNew())
			if o == nil {
				t.Fatal("should have gotten default")
			}
		})
	})

	// String conversion
	t.Run("AsString", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			v := ValueNew("foo")
			_ = v.AsString()
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew(1)
			_, err := try.Apply(v.AsString)
			if err == nil {
				t.Fatal("conversion should have failed")
			}
		})
	})
	t.Run("IsString", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			v := ValueNew("bar")
			if !v.IsString() {
				t.Fatal("Value is a string")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew(1)
			if v.IsString() {
				t.Fatal("Value is not a string")
			}
		})
	})
	t.Run("ToString", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			v := ValueNew("bar")
			o := v.ToString()
			if !equal(o, "bar") {
				t.Fatal("didn't get expected result")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew(1)
			o := v.ToString()
			if o != "" {
				t.Fatal("Value should not be a string")
			}
		})
		t.Run("Default", func(t *testing.T) {
			v := ValueNew(-1)
			o := v.ToString("bar")
			if o != "bar" {
				t.Fatal("should have gotten default")
			}
		})
	})

	// int32 conversion
	t.Run("AsInt32", func(t *testing.T) {
		t.Run("Int32", func(t *testing.T) {
			v := ValueNew(int32(10))
			_ = v.AsInt32()
		})
		t.Run("Convertible", func(t *testing.T) {
			v := ValueNew(int64(10))
			_ = v.AsInt32()
			v = ValueNew(float64(10))
			_ = v.AsInt32()
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			_, err := try.Apply(v.AsInt32)
			if err == nil {
				t.Fatal("conversion should have failed")
			}
		})
	})
	t.Run("IsInt32", func(t *testing.T) {
		t.Run("Int32", func(t *testing.T) {
			v := ValueNew(int32(10))
			if !v.IsInt32() {
				t.Fatal("Value is a int32")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew(int64(10))
			if v.IsInt32() {
				t.Fatal("Value is not a int32")
			}
		})
	})
	t.Run("ToInt32", func(t *testing.T) {
		t.Run("Int32", func(t *testing.T) {
			v := ValueNew(int32(-1))
			o := v.ToInt32()
			if !equal(o, int32(-1)) {
				t.Fatal("didn't get expected result", o)
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToInt32()
			if o != 0 {
				t.Fatal("Value should not be a int32")
			}
		})
		t.Run("Default", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToInt32(10)
			if o != 10 {
				t.Fatal("should have gotten default")
			}
		})
	})

	// uint32 conversion
	t.Run("AsUint32", func(t *testing.T) {
		t.Run("Uint32", func(t *testing.T) {
			v := ValueNew(uint32(10))
			_ = v.AsUint32()
		})
		t.Run("Convertible", func(t *testing.T) {
			v := ValueNew(int64(10))
			_ = v.AsUint32()
			v = ValueNew(float64(10))
			_ = v.AsUint32()
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			_, err := try.Apply(v.AsUint32)
			if err == nil {
				t.Fatal("conversion should have failed")
			}
		})
	})
	t.Run("IsUint32", func(t *testing.T) {
		t.Run("Uint32", func(t *testing.T) {
			v := ValueNew(uint32(10))
			if !v.IsUint32() {
				t.Fatal("Value is a uint32")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew(int64(10))
			if v.IsUint32() {
				t.Fatal("Value is not a uint32")
			}
		})
	})
	t.Run("ToUint32", func(t *testing.T) {
		t.Run("Uint32", func(t *testing.T) {
			v := ValueNew(uint32(10))
			o := v.ToUint32()
			if !equal(o, uint32(10)) {
				t.Fatal("didn't get expected result", o)
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToUint32()
			if o != 0 {
				t.Fatal("Value should not be a uint32")
			}
		})
		t.Run("Default", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToUint32(10)
			if o != 10 {
				t.Fatal("should have gotten default")
			}
		})
	})

	// int64 conversion
	t.Run("AsInt64", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			v := ValueNew(int64(10))
			_ = v.AsInt64()
		})
		t.Run("Convertible", func(t *testing.T) {
			v := ValueNew(int32(10))
			_ = v.AsInt64()
			v = ValueNew(float64(10))
			_ = v.AsInt64()
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			_, err := try.Apply(v.AsInt64)
			if err == nil {
				t.Fatal("conversion should have failed")
			}
		})
	})
	t.Run("IsInt64", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			v := ValueNew(int64(10))
			if !v.IsInt64() {
				t.Fatal("Value is a int64")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew(int32(10))
			if v.IsInt64() {
				t.Fatal("Value is not a int64")
			}
		})
	})
	t.Run("ToInt64", func(t *testing.T) {
		t.Run("Int64", func(t *testing.T) {
			v := ValueNew(int64(-1))
			o := v.ToInt64()
			if !equal(o, int64(-1)) {
				t.Fatal("didn't get expected result", o)
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToInt64()
			if o != 0 {
				t.Fatal("Value should not be a int64")
			}
		})
		t.Run("Default", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToInt64(10)
			if o != 10 {
				t.Fatal("should have gotten default")
			}
		})
	})

	// uint64 conversion
	t.Run("AsUint64", func(t *testing.T) {
		t.Run("Uint64", func(t *testing.T) {
			v := ValueNew(uint64(10))
			_ = v.AsUint64()
		})
		t.Run("Convertible", func(t *testing.T) {
			v := ValueNew(int32(10))
			_ = v.AsUint64()
			v = ValueNew(float64(10))
			_ = v.AsUint64()
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			_, err := try.Apply(v.AsUint64)
			if err == nil {
				t.Fatal("conversion should have failed")
			}
		})
	})
	t.Run("IsUint64", func(t *testing.T) {
		t.Run("Uint64", func(t *testing.T) {
			v := ValueNew(uint64(10))
			if !v.IsUint64() {
				t.Fatal("Value is a uint64")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			if v.IsUint64() {
				t.Fatal("Value is not a uint64")
			}
		})
	})
	t.Run("ToUint64", func(t *testing.T) {
		t.Run("Uint64", func(t *testing.T) {
			v := ValueNew(uint64(10))
			o := v.ToUint64()
			if !equal(o, uint64(10)) {
				t.Fatal("didn't get expected result", o)
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToUint64()
			if o != 0 {
				t.Fatal("Value should not be a uint64")
			}
		})
		t.Run("Default", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToUint64(10)
			if o != 10 {
				t.Fatal("should have gotten default")
			}
		})
	})

	// float conversion
	t.Run("AsFloat", func(t *testing.T) {
		t.Run("Float", func(t *testing.T) {
			v := ValueNew(float64(10))
			_ = v.AsFloat()
		})
		t.Run("Convertible", func(t *testing.T) {
			v := ValueNew(int32(10))
			_ = v.AsFloat()
			v = ValueNew(uint64(10))
			_ = v.AsFloat()
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			_, err := try.Apply(v.AsFloat)
			if err == nil {
				t.Fatal("conversion should have failed")
			}
		})
	})
	t.Run("IsFloat", func(t *testing.T) {
		t.Run("Float", func(t *testing.T) {
			v := ValueNew(float64(10))
			if !v.IsFloat() {
				t.Fatal("Value is a float")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			if v.IsFloat() {
				t.Fatal("Value is not a float")
			}
		})
	})
	t.Run("ToFloat", func(t *testing.T) {
		t.Run("Float", func(t *testing.T) {
			v := ValueNew(float64(10))
			o := v.ToFloat()
			if !equal(o, float64(10)) {
				t.Fatal("didn't get expected result", o)
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToFloat()
			if o != 0 {
				t.Fatal("Value should not be a float")
			}
		})
		t.Run("Default", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToFloat(10)
			if o != 10 {
				t.Fatal("should have gotten default")
			}
		})
	})

	// boolean conversion
	t.Run("AsBoolean", func(t *testing.T) {
		t.Run("Bool", func(t *testing.T) {
			v := ValueNew(false)
			_ = v.AsBoolean()
		})
		t.Run("Empty", func(t *testing.T) {
			v := Empty()
			_ = v.AsBoolean()
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			_, err := try.Apply(v.AsBoolean)
			if err == nil {
				t.Fatal("conversion should have failed")
			}
		})
	})
	t.Run("IsBoolean", func(t *testing.T) {
		t.Run("Bool", func(t *testing.T) {
			v := ValueNew(false)
			if !v.IsBoolean() {
				t.Fatal("should have been boolean")
			}
		})
		t.Run("Empty", func(t *testing.T) {
			v := Empty()
			if !v.IsBoolean() {
				t.Fatal("should have been boolean")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			if v.IsBoolean() {
				t.Fatal("should not have been boolean")
			}
		})
	})
	t.Run("ToBoolean", func(t *testing.T) {
		t.Run("Boolean", func(t *testing.T) {
			v := ValueNew(false)
			o := v.ToBoolean()
			if o {
				t.Fatal("didn't get expected result", o)
			}
		})
		t.Run("Empty", func(t *testing.T) {
			v := Empty()
			o := v.ToBoolean()
			if !o {
				t.Fatal("didn't get expected result", o)
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToBoolean()
			if o {
				t.Fatal("Value should not be a bool")
			}
		})
		t.Run("Default", func(t *testing.T) {
			v := ValueNew("foo")
			o := v.ToBoolean(true)
			if !o {
				t.Fatal("should have gotten default")
			}
		})
	})

	// instance-identifier conversion
	t.Run("AsInstanceID", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			v := ValueNew("/foo:bar/baz")
			_ = v.AsInstanceID()
		})
		t.Run("InstanceID", func(t *testing.T) {
			v := ValueNew(InstanceIDNew("/foo:bar/baz"))
			_ = v.AsInstanceID()
		})
		t.Run("String-invalid", func(t *testing.T) {
			v := ValueNew("/foo/bar")
			_, err := try.Apply(v.AsInstanceID)
			if err == nil {
				t.Fatal("should have failed")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew(10)
			_, err := try.Apply(v.AsInstanceID)
			if err == nil {
				t.Fatal("should have failed")
			}
		})
	})
	t.Run("IsInstanceID", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			v := ValueNew("/foo:bar/baz")
			if !v.IsInstanceID() {
				t.Fatal("should have been an instanceID")
			}
		})
		t.Run("InstanceID", func(t *testing.T) {
			v := ValueNew(InstanceIDNew("/foo:bar/baz"))
			if !v.IsInstanceID() {
				t.Fatal("should have been an instanceID")
			}
		})
		t.Run("String-invalid", func(t *testing.T) {
			v := ValueNew("/foo/bar")
			if v.IsInstanceID() {
				t.Fatal("should not have been an instanceID")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew(10)
			if v.IsInstanceID() {
				t.Fatal("should not have been an instanceID")
			}
		})
	})
	t.Run("ToInstanceID", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			v := ValueNew("/foo:bar/baz")
			iid := v.ToInstanceID()
			if iid == nil {
				t.Fatal("should have been an instanceID")
			}
		})
		t.Run("InstanceID", func(t *testing.T) {
			v := ValueNew(InstanceIDNew("/foo:bar/baz"))
			iid := v.ToInstanceID()
			if iid == nil {
				t.Fatal("should have been an instanceID")
			}
		})
		t.Run("String-invalid", func(t *testing.T) {
			v := ValueNew("/foo/bar")
			iid := v.ToInstanceID()
			if iid != nil {
				t.Fatal("should not have been an instanceID")
			}
		})
		t.Run("Other", func(t *testing.T) {
			v := ValueNew(10)
			iid := v.ToInstanceID()
			if iid != nil {
				t.Fatal("should not have been an instanceID")
			}
		})
		t.Run("Default", func(t *testing.T) {
			v := ValueNew(10)
			iid := v.ToInstanceID(InstanceIDNew("/foo:bar/baz"))
			if iid == nil {
				t.Fatal("should have been an instanceID")
			}
		})
	})

	// Native conversions
	t.Run("ToInterface", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			v := ValueNew("foo")
			if v.ToInterface() != v.data {
				t.Fatal("ToInterface should yeild the data untouched")
			}
		})
		t.Run("Object", func(t *testing.T) {
			v := ValueNew(ObjectWith(PairNew("foo", "bar")))
			if v.ToInterface() != v.data {
				t.Fatal("ToInterface should yeild the data untouched")
			}
		})
		t.Run("Array", func(t *testing.T) {
			v := ValueNew(ArrayWith(1, 2, 4, 5))
			if v.ToInterface() != v.data {
				t.Fatal("ToInterface should yeild the data untouched")
			}
		})
	})
	t.Run("ToData", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			v := ValueNew("foo")
			if v.ToData() != v.data {
				t.Fatal("ToData should yeild the data untouched")
			}
		})
		t.Run("Object", func(t *testing.T) {
			v := ValueNew(ObjectWith(PairNew("foo", "bar")))
			d := v.ToData()
			_ = d.(map[string]*Value)
		})
		t.Run("Array", func(t *testing.T) {
			v := ValueNew(ArrayWith(1, 2, 4, 5))
			d := v.ToData()
			_ = d.([]*Value)
		})

	})
	t.Run("ToNative", func(t *testing.T) {
		t.Run("String", func(t *testing.T) {
			v := ValueNew("foo")
			if v.ToNative() != v.data {
				t.Fatal("ToData should yeild the data untouched")
			}
		})
		t.Run("Object", func(t *testing.T) {
			v := ValueNew(ObjectWith(PairNew("foo", "bar")))
			d := v.ToNative()
			_ = d.(map[string]interface{})
		})
		t.Run("Array", func(t *testing.T) {
			v := ValueNew(ArrayWith(1, 2, 4, 5))
			d := v.ToNative()
			_ = d.([]interface{})
		})
		t.Run("Empty", func(t *testing.T) {
			v := Empty()
			d := v.ToNative()
			_ = d.([]interface{})
		})
	})

	// Special type checks
	t.Run("IsEmpty", func(t *testing.T) {
		t.Run("from native", func(t *testing.T) {
			v := ValueNew([]interface{}{nil})
			if !v.IsEmpty() {
				t.Fatal("empty construction failed")
			}
		})
		t.Run("from rfc7951", func(t *testing.T) {
			v := ValueNew(nil)
			data := "[null]"
			err := rfc7951.Unmarshal([]byte(data), v)
			if err != nil {
				t.Fatal(err)
			}
			if !v.IsEmpty() {
				t.Fatal("empty construction failed")
			}
		})
	})
	t.Run("IsNull", func(t *testing.T) {
		if !ValueNew(nil).IsNull() {
			t.Fatal("should have been null")
		}
	})
}

func ExampleValue_ToData() {
	tree := TreeFromObject(TESTOBJ)
	const test = `
{{- range (.At "/module-v1:nested/list").ToData -}}
{{with .ToObject -}}
{{.At "key"}} {{.At "objleaf"}}
{{end -}}
{{end -}}
{{range (.At "/module-v1:nested/leaf-list").ToData -}}
{{.}}
{{end -}}
`
	testTmpl := template.Must(template.New("test").Parse(test))
	testTmpl.Execute(os.Stdout, tree)
	// Output: foo bar
	// bar baz
	// baz quux
	// quux quuz
	// 1
	// 2
	// 3
	// 4
	// 5
	// 6
	// 7
}
