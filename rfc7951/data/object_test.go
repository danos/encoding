// Copyright (c) 2018-2020, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"bytes"
	"reflect"
	"strconv"
	"testing"

	"github.com/danos/encoding/rfc7951"
)

func testCollectionObject(cons func(sz int) *Object, t *testing.T) {
	t.Run("At/coll.Assoc(X,Y);coll.At(X)==Y",
		func(t *testing.T) {
			coll := cons(1)
			index := "0"
			val := 10
			coll = coll.Assoc(index, val)
			got := coll.At(index)
			assert(equal(got, ValueNew(val)), func() {
				t.Fatalf("expected %v, got %v\n", val, got)
			})
			coll = cons(4)
			index = "3"
			val = 10
			coll = coll.Assoc(index, val)
			got = coll.At(index)
			assert(equal(got, ValueNew(val)), func() {
				t.Fatalf("expected %v, got %v\n", val, got)
			})
		})
	t.Run("Assoc/coll.Assoc(X,Y).At(X)==Y", func(t *testing.T) {
		coll := cons(1)
		index := "0"
		val := 10
		coll = coll.Assoc(index, val)
		got := coll.At(index)
		assert(equal(got, ValueNew(val)), func() {
			t.Fatalf("expected %v, got %v\n", val, got)
		})
	})
	t.Run("Do", func(t *testing.T) {
		var expCount, count int32
		coll := cons(100)
		for i := 0; i < 100; i++ {
			coll.Assoc(strconv.Itoa(i), i)
			expCount += int32(i)
		}
		coll.Range(func(elem *Value) { count += elem.AsInt32() })
		assert(count == expCount, func() {
			t.Fatalf("expected %v, got %v\n", expCount, count)
		})
	})
	t.Run("Length/sz:=coll.Length();coll.Assoc(X);coll.Length()==sz+1",
		func(t *testing.T) {
			coll := cons(0)
			sz := coll.Length()
			coll = coll.Assoc("1", 1)
			assert(coll.Length() == sz+1, func() {
				t.Fatalf("expected %v, got %v\n", sz+1,
					coll.Length())
			})
		})
	t.Run("KeysDo", func(t *testing.T) {
		sum := 0
		cons(3).Range(func(key string) {
			k, _ := strconv.Atoi(key)
			sum += k
		})
		assert(sum == 3, func() {
			t.Fatalf("expected %v, got %v\n", 3,
				sum)
		})
	})
	t.Run("ValuesDo", func(t *testing.T) {
		sum := int32(0)
		cons(3).Range(func(val *Value) {
			sum += val.AsInt32()
		})
		assert(sum == 3, func() {
			t.Fatalf("expected %v, got %v\n", 3,
				sum)
		})
	})
	t.Run("PairsDo", func(t *testing.T) {
		cons(3).Range(func(assoc Pair) {
			if assoc.Key() != strconv.Itoa(int(assoc.Value().AsInt32())) {
				t.Fatal("key and value should match")
			}
		})
	})
	t.Run("Delete", func(t *testing.T) {
		sz := cons(2).Delete("1").Length()
		assert(sz == 1, func() {
			t.Fatalf("expected %v, got %v\n", 1, sz)
		})
	})
	t.Run("Delete non-existent", func(t *testing.T) {
		sz := cons(2).Delete("4").Length()
		assert(sz == 2, func() {
			t.Fatalf("expected %v, got %v\n", 1, sz)
		})
	})
}

func testEqualObjects(t *testing.T, c1, c2 *Object) {
	c1.Range(func(key string, value *Value) {
		if !c2.Contains(key) {
			t.Fatal("expected element not found in c2", key)
		}
		if !equal(value, c2.At(key)) {
			t.Fatal("expected element not found in c2", key, value)
		}

	})
	c2.Range(func(key string, value *Value) {
		if !c1.Contains(key) {
			t.Fatal("expected element not found in c1", key)
		}
		if !equal(value, c1.At(key)) {
			t.Fatal("expected element not found in c1", key, value)
		}
	})
}

func TestCollectionSemanticsObject(t *testing.T) {
	testCollectionObject(func(sz int) *Object {
		out := ObjectNew()
		for i := 0; i < sz; i++ {
			out = out.Assoc(strconv.Itoa(i), i)
		}
		return out
	}, t)
}

func TestObjectNewWithPairs(t *testing.T) {
	coll := ObjectWith(
		PairNew("1", 2),
		PairNew("3", 4),
		PairNew("5", 6))
	fatal := func(exp, got interface{}) func() {
		return func() {
			t.Fatalf("expected %v, got %v\n", exp, got)
		}
	}
	assert(coll.At("1").AsInt32() == 2, fatal(coll.At("1"), 2))
	assert(coll.At("3").AsInt32() == 4, fatal(coll.At("3"), 4))
	assert(coll.At("5").AsInt32() == 6, fatal(coll.At("5"), 6))
}

func TestObjectNewFrom(t *testing.T) {
	coll := ObjectFrom(map[string]interface{}{
		"1": 2,
		"3": 4,
		"5": 6,
	})
	fatal := func(exp, got interface{}) func() {
		return func() {
			t.Fatalf("expected %v, got %v\n", exp, got)
		}
	}
	assert(coll.At("1").AsInt32() == 2, fatal(coll.At("1"), 2))
	assert(coll.At("3").AsInt32() == 4, fatal(coll.At("3"), 4))
	assert(coll.At("5").AsInt32() == 6, fatal(coll.At("5"), 6))
}

func TestObjectKeysDo(t *testing.T) {
	coll := ObjectFrom(map[string]interface{}{
		"1": 2,
		"3": 4,
		"5": 6,
	})
	coll.Range(func(key string) {
		k, _ := strconv.Atoi(key)
		if k%2 == 0 {
			t.Fatal("keys should be odd")
		}
	})
}

func TestObjectValuesDo(t *testing.T) {
	coll := ObjectFrom(map[string]interface{}{
		"1": 2,
		"3": 4,
		"5": 6,
	})
	coll.Range(func(val *Value) {
		if val.AsInt32()%2 != 0 {
			t.Fatal("vals should be even")
		}
	})
}

func TestObjectPairsDo(t *testing.T) {
	coll := ObjectFrom(map[string]interface{}{
		"1": 2,
		"3": 4,
		"5": 6,
	})
	coll.Range(func(assoc Pair) {
		k, _ := strconv.Atoi(assoc.Key())
		if k%2 == 0 {
			t.Fatal("keys should be odd")
		}
		if assoc.Value().AsInt32()%2 != 0 {
			t.Fatal("vals should be even")
		}
	})
}

func TestObjectEquiv(t *testing.T) {
	/* Create the following object 3 ways and ensure they are equivalent
	 * {
	 *	"module-v1:foo": {
	 *		"bar": {
	 *			"baz":["quux","foo"],
	 *			"quux":"quuz"
	 *		},
	 *		"baz":"quux"
	 *	},
	 *	"module-v1:bar":"baz"
	 * }
	 */
	one := ObjectWith(
		PairNew("module-v1:foo", ObjectWith(
			PairNew("bar", ObjectWith(
				PairNew("baz", ArrayWith("quux", "foo")),
				PairNew("quux", "quuz"))),
			PairNew("baz", "quux"))),
		PairNew("module-v1:bar", "baz"),
		PairNew("module-v2:baz", ArrayWith(
			ObjectWith(
				PairNew("quux", "foo"),
				PairNew("baz", "bar")),
			ObjectWith(
				PairNew("quux", "bar"),
				PairNew("baz", "foo")))))
	two := ObjectFrom(map[string]interface{}{
		"module-v1:foo": ObjectFrom(map[string]interface{}{
			"bar": ObjectFrom(map[string]interface{}{
				"baz":  ArrayFrom([]interface{}{"quux", "foo"}),
				"quux": "quuz",
			}),
			"baz": "quux",
		}),
		"module-v1:bar": "baz",
		"module-v2:baz": ArrayFrom([]interface{}{
			ObjectFrom(map[string]interface{}{
				"quux": "foo",
				"baz":  "bar",
			}),
			ObjectFrom(map[string]interface{}{
				"quux": "bar",
				"baz":  "foo",
			}),
		}),
	})
	three := ObjectFrom(map[string]interface{}{
		"module-v1:foo": map[string]interface{}{
			"module-v1:bar": map[string]interface{}{
				"baz":  []interface{}{"quux", "foo"},
				"quux": "quuz",
			},
			"baz": "quux",
		},
		"module-v1:bar": "baz",
		"module-v2:baz": []interface{}{
			map[string]interface{}{
				"quux": "foo",
				"baz":  "bar",
			},
			map[string]interface{}{
				"quux": "bar",
				"baz":  "foo",
			},
		},
	})
	t.Run("equivalent", func(t *testing.T) {
		if !equal(one, two) || !equal(two, three) {
			t.Fatalf("equivalent object creation mechanisms should always yeild the same object\n one: %s\n\ntwo:%s\n\nthree:%s", one, two, three)

		}
	})

	t.Run("correct-namespaces", func(t *testing.T) {
		correct := map[string]interface{}{
			"module-v1:foo": map[string]interface{}{
				"module-v1:bar": map[string]interface{}{
					"module-v1:baz":  []interface{}{"quux", "foo"},
					"module-v1:quux": "quuz",
				},
				"module-v1:baz": "quux",
			},
			"module-v1:bar": "baz",
			"module-v2:baz": []interface{}{
				map[string]interface{}{
					"module-v2:quux": "foo",
					"module-v2:baz":  "bar",
				},
				map[string]interface{}{
					"module-v2:quux": "bar",
					"module-v2:baz":  "foo",
				},
			},
		}
		if !reflect.DeepEqual(correct, one.toNative()) {
			t.Fatal("native object is not equivalent to a properly annotated tree")

		}
	})

}

func TestCanAccessWithImplicitOrExplicitModuleName(t *testing.T) {
	one := ObjectWith(
		PairNew("module-v1:foo", ObjectWith(
			PairNew("bar", ObjectWith(
				PairNew("baz", ArrayWith("quux", "foo")),
				PairNew("quux", "quuz"))),
			PairNew("baz", "quux"),
			PairNew("v2:zzz", "abc"))),
		PairNew("module-v1:bar", "baz"),
		PairNew("module-v2:baz", ArrayWith(
			ObjectWith(
				PairNew("quux", "foo"),
				PairNew("baz", "bar")),
			ObjectWith(
				PairNew("quux", "bar"),
				PairNew("baz", "foo")))))
	if one.At("module-v1:foo").AsObject().
		At("bar").AsObject().
		At("quux").RFC7951String() != "quuz" {
		t.Fatal("implicit module access failed")
	}
	if one.At("module-v1:foo").AsObject().
		At("module-v1:bar").AsObject().
		At("module-v1:quux").RFC7951String() != "quuz" {
		t.Fatal("explicit module access failed")
	}
}

func TestObjectMarshalRFC7951(t *testing.T) {
	obj := ObjectFrom(map[string]interface{}{
		"module-v1:foo": map[string]interface{}{
			"module-v1:bar": map[string]interface{}{
				"baz":  []interface{}{"quux", "foo"},
				"quux": "quuz",
			},
			"baz":                       "quux",
			"one":                       1,
			"two.one":                   2.1,
			"true":                      true,
			"false":                     false,
			"empty":                     []interface{}{nil},
			"nil":                       nil,
			"negative":                  -2,
			"negative-float":            "-2.4",
			"negative-in-dotted-string": "-2.fooboar",
			"negative-in-string":        "-foobar",
			"dotted-string":             "192.168.1.1/24",
			"uint64":                    "1234",
			"negative-uint64":           "-1234",
			"positive":                  "+2",
			"positive-float":            "+2.3",
			"plus-in-string":            "+foobar",
			"plus-in-dotted-string":     "+2.foobar",
			"empty-string":              "",
		},
		"module-v1:bar": "baz",
		"module-v2:baz": []interface{}{
			map[string]interface{}{
				"quux": "foo",
				"baz":  "bar",
			},
			map[string]interface{}{
				"quux": "bar",
				"baz":  "foo",
			},
		},
	})
	v := ValueNew(obj)
	var buf bytes.Buffer
	v.marshalRFC7951(&buf, "")
	o := objectNew()
	o.unmarshalRFC7951(buf.Bytes(), "",
		stringInternerNew(), valueInternerNew())
	got := ValueNew(o)
	expected := `{"module-v1:bar":"baz","module-v2:baz":[{"quux":"foo","baz":"bar"},{"quux":"bar","baz":"foo"}],"module-v1:foo":{"negative-uint64":"-1234","nil":null,"false":false,"plus-in-string":"+foobar","true":true,"empty":[null],"two.one":"2.1","negative-in-dotted-string":"-2.fooboar","negative":-2,"bar":{"quux":"quuz","baz":["quux","foo"]},"negative-in-string":"-foobar","plus-in-dotted-string":"+2.foobar","negative-float":"-2.4","baz":"quux","positive-float":"+2.3","one":1,"empty-string":"","dotted-string":"192.168.1.1/24","positive":"2","uint64":"1234"}}`
	tree := TreeNew()
	rfc7951.Unmarshal([]byte(expected), tree)
	if !equal(tree.Root(), got) {
		var gotbuf bytes.Buffer
		got.marshalRFC7951(&gotbuf, "")
		t.Fatalf("got %s, expected %s\n", gotbuf.String(), expected)
	}
}

func TestEscapedStringMarshalRFC7951(t *testing.T) {
	obj := ObjectFrom(map[string]interface{}{
		"module-v1:foo": map[string]interface{}{
			"empty-string":        "",
			"one-quote":           "\"",
			"quotes-in-string":    "\"foo\" \"bar\"",
			"backslash-in-string": "\\foo\\bar",
			"newline-in-string":   "foo\nbar",
			"tab-in-string":       "\tfoo\tbar",
		},
		"module-v2:baz": []interface{}{
			map[string]interface{}{
				"quux": "\"foo\"",
				"baz":  "bar",
			},
			map[string]interface{}{
				"quux": "\"bar\"",
				"baz":  "foo",
			},
		},
	})
	v := ValueNew(obj)
	var buf bytes.Buffer
	v.marshalRFC7951(&buf, "")
	o := objectNew()
	o.unmarshalRFC7951(buf.Bytes(), "",
		stringInternerNew(), valueInternerNew())
	got := ValueNew(o)
	expected := `{"module-v2:baz":[{"quux":"\"foo\"","baz":"bar"},{"quux":"\"bar\"","baz":"foo"}],"module-v1:foo":{"empty-string":"","one-quote":"\"","quotes-in-string":"\"foo\" \"bar\"","backslash-in-string":"\\foo\\bar","newline-in-string":"foo\nbar","tab-in-string":"\tfoo\tbar"}}`
	tree := TreeNew()
	rfc7951.Unmarshal([]byte(expected), tree)
	eobj := tree.Root()
	if !equal(eobj, got) {
		var gotbuf bytes.Buffer
		got.marshalRFC7951(&gotbuf, "")
		t.Fatalf("got:\n\t%s\n\nexpected:\n\t%s\n", gotbuf.String(), expected)
	}
}

func TestPair(t *testing.T) {
	t.Run("Pair equality", func(t *testing.T) {
		p1, p2, p3 :=
			PairNew("a", "b"), PairNew("a", "b"), PairNew("a", "c")
		if !equal(p1, p2) {
			t.Fatal(p1, "!=", p2)
		}
		if equal(p2, p3) {
			t.Fatal(p2, "==", p3)
		}
		if equal(p1, "foo") {
			t.Fatal(p2, "==", "foo")
		}
	})
	t.Run("Pair String", func(t *testing.T) {
		p1 := PairNew("a", "b")
		if p1.String() != "[a b]" {
			t.Fatal(p1.String(), "isn't as expected")
		}
	})
}

func TestObjectFind(t *testing.T) {
	obj := TESTOBJ
	t.Run("existing key", func(t *testing.T) {
		v, ok := obj.Find("module-v1:container")
		if !ok || v == nil {
			t.Fatal("didn't find expected value")
		}
	})
	t.Run("non-existant key", func(t *testing.T) {
		v, ok := obj.Find("container")
		if ok || v != nil {
			t.Fatal("found unexpected value")
		}
	})
}

func TestObjectToData(t *testing.T) {
	obj := ObjectWith(PairNew("a", "b"),
		PairNew("c", "d"),
		PairNew("e", "f"))
	data := obj.toData()
	for k, v := range data.(map[string]*Value) {
		if !equal(obj.At(k), v) {
			t.Fatal("data didn't convert to exact copy")
		}
	}
}

func TestObjectString(t *testing.T) {
	str := TESTOBJ.String()
	tree := TreeNew()
	err := rfc7951.Unmarshal([]byte(str), tree)
	if err != nil {
		t.Fatal(err)
	}
	orig := TreeFromObject(TESTOBJ)
	if !equal(tree, orig) {
		t.Fatalf("got:\n\t%s\nexpected:\n\t%s\ndifferences:\n\t%s\n",
			tree,
			orig,
			tree.Diff(orig))
	}
}

func TestTObject(t *testing.T) {
	t.Run("At", func(t *testing.T) {
		TESTOBJ.Transform(func(obj *TObject) {
			if obj.At("module-v1:leaf").String() != "foo" {
				t.Fatal("didn't retrieve expected value")
			}
			if obj.At("module-v2:leaf") != nil {
				t.Fatal("didn't retrieve expected value")
			}
		})
	})
	t.Run("Assoc", func(t *testing.T) {
		new := TESTOBJ.Transform(func(obj *TObject) {
			obj.Assoc("module-v1:leaf", "bar")
		})
		if new.At("module-v1:leaf") == TESTOBJ.At("module-v1:leaf") {
			t.Fatal("object updated incorrectly")
		}
		if new.At("module-v1:leaf").String() != "bar" {
			t.Fatal("object didn't update correctly")
		}

	})
	t.Run("Contains", func(t *testing.T) {
		TESTOBJ.Transform(func(obj *TObject) {
			if !obj.Contains("module-v1:leaf") {
				t.Fatal("didn't find expected value")
			}
			if obj.Contains("module-v2:leaf") {
				t.Fatal("found invalid value")
			}
		})
	})
	t.Run("Delete", func(t *testing.T) {
		new := TESTOBJ.Transform(func(obj *TObject) {
			obj.Delete("module-v1:leaf")
		})
		if new.Contains("module-v1:leaf") {
			t.Fatal("delete failed to remove value")
		}
	})
	t.Run("Equal", func(t *testing.T) {
		TESTOBJ.Transform(func(obj1 *TObject) {
			TESTOBJ.Transform(func(obj2 *TObject) {
				if !obj1.Equal(obj2) {
					t.Fatal("object not equal to its self")
				}
			})
		})
		TESTOBJ.Transform(func(obj1 *TObject) {
			obj1.Delete("module-v1:leaf")
			TESTOBJ.Transform(func(obj2 *TObject) {
				if obj1.Equal(obj2) {
					t.Fatal("object equal to different object")
				}
			})
		})
	})
	t.Run("Find", func(t *testing.T) {
		TESTOBJ.Transform(func(obj *TObject) {
			v, ok := obj.Find("module-v1:leaf")
			if !ok || v.String() != "foo" {
				t.Fatal("didn't find expected value")
			}
			_, ok = obj.Find("module-v2:leaf")
			if ok {
				t.Fatal("found invalid value")
			}
		})
	})
	t.Run("Length", func(t *testing.T) {
		TESTOBJ.Transform(func(obj *TObject) {
			if obj.Length() != TESTOBJ.Length() {
				t.Fatal("length of transient object not as expected")
			}
		})
	})
}
