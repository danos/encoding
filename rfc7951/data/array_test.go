// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"strconv"
	"testing"
	"unicode"

	"jsouthworth.net/go/dyn"
)

func testCollectionArray(cons func(sz int) *Array, t *testing.T) {
	t.Run("At/coll.Assoc(X,Y);coll.At(X)==Y",
		func(t *testing.T) {
			coll := cons(1)
			index := 0
			val := 10
			coll = coll.Assoc(index, val)
			got := coll.At(index)
			assert(equal(got, ValueNew(val)), func() {
				t.Fatalf("expected %v, got %v\n", val, got)
			})
			coll = cons(4)
			index = 3
			val = 10
			coll = coll.Assoc(index, val)
			got = coll.At(index)
			assert(equal(got, ValueNew(val)), func() {
				t.Fatalf("expected %v, got %v\n", val, got)
			})
		})
	t.Run("At/coll.At(inval)returns nil",
		func(t *testing.T) {
			coll := cons(1)
			index := 2
			assert(coll.At(index) == nil, func() {
				t.Fatal("should have returned nil")
			})
		})
	t.Run("Assoc/coll.Assoc(X,Y).At(X)==Y", func(t *testing.T) {
		coll := cons(1)
		index := 0
		val := 10
		coll = coll.Assoc(index, val)
		got := coll.At(index)
		assert(equal(got, ValueNew(val)), func() {
			t.Fatalf("expected %v, got %v\n", val, got)
		})
	})
	t.Run("detect/coll.Assoc(0,1);coll.Detect(1)==true", func(t *testing.T) {
		coll := cons(1)
		coll = coll.Assoc(0, 1)
		assert(coll.detect(func(elem *Value) bool {
			return elem.AsInt32() == 1
		}) != nil, func() { t.Fatal("expected element not found") })
	})
	t.Run("Detect/coll.detect(non-exist)==false", func(t *testing.T) {
		assert(cons(0).detect(func(elem *Value) bool {
			return elem.AsInt32() == 1
		}) == nil, func() { t.Fatal("unexpected element found") })
	})
	t.Run("detectAndIfNone/custom", func(t *testing.T) {
		coll := cons(1)
		coll = coll.Assoc(0, 1)
		assert(coll.detectAndIfNone(
			func(elem *Value) bool {
				return elem.AsInt32() == 1
			},
			func() *Value {
				return ValueNew(10)
			}).AsInt32() != 10,
			func() {
				t.Fatal("expected element not returned")
			})
	})
	t.Run("Range", func(t *testing.T) {
		var expCount, count int32
		coll := cons(100)
		for i := 0; i < 100; i++ {
			coll = coll.Assoc(i, i)
			expCount += int32(i)
		}
		coll.Range(func(elem *Value) { count += elem.AsInt32() })
		assert(count == expCount, func() {
			t.Fatalf("expected %v, got %v\n", expCount, count)
		})
	})
	t.Run("selectItems/only selects matching elements", func(t *testing.T) {
		coll := cons(0)
		expEvens := cons(0)
		for i := 0; i < 10; i++ {
			if i%2 == 0 {
				expEvens = expEvens.Append(i)
			}
			coll = coll.Append(i)
		}
		evens := coll.selectItems(func(elem *Value) bool {
			return elem.AsInt32()%2 == 0
		})
		testEqualArrays(t, expEvens, evens)
	})
	t.Run("Length/sz:=coll.Length();coll.Append(X);coll.Length()==sz+1",
		func(t *testing.T) {
			coll := cons(0)
			sz := coll.Length()
			coll = coll.Append(1)
			assert(coll.Length() == sz+1, func() {
				t.Fatalf("expected %v, got %v\n", sz+1,
					coll.Length())
			})
		})
	t.Run("KeysDo", func(t *testing.T) {
		sum := 0
		cons(0).Append(0).Append(1).Append(2).
			Range(func(key int) {
				sum += key
			})
		assert(sum == 3, func() {
			t.Fatalf("expected %v, got %v\n", 3,
				sum)
		})
	})
	t.Run("ValuesDo", func(t *testing.T) {
		sum := int32(0)
		cons(0).Append(0).Append(1).Append(2).
			Range(func(val *Value) {
				sum += val.AsInt32()
			})
		assert(sum == 3, func() {
			t.Fatalf("expected %v, got %v\n", 3,
				sum)
		})
	})
	t.Run("RemoveAt", func(t *testing.T) {
		sz := cons(0).Append(0).Append(1).Delete(1).Length()
		assert(sz == 1, func() {
			t.Fatalf("expected %v, got %v\n", 1, sz)
		})
	})
}

func TestCollectionSemanticsArray(t *testing.T) {
	testCollectionArray(func(sz int) *Array {
		coll := ArrayNew()
		for i := 0; i < sz; i++ {
			coll = coll.Append(nil)
		}
		return coll
	}, t)
}

func TestArrayNewWith(t *testing.T) {
	array := ArrayWith(0, 1, 2, 3, 4)
	for i := 0; i < 5; i++ {
		if array.At(i).AsInt64() != int64(i) {
			t.Fatal("expected value not found")
		}
	}
}

func TestArraySpecifics(t *testing.T) {
	t.Run("At/coll.Append(X);coll.At(coll.Length()-1)==X",
		func(t *testing.T) {
			coll := ArrayNew()
			val := "foo"
			coll = coll.Append(val)
			got := coll.At(coll.Length() - 1)
			assert(got.RFC7951String() == val, func() {
				t.Fatalf("expected %v, got %v\n", val, got)
			})
		})
}

func testEqualArrays(t *testing.T, c1, c2 *Array) {
	c1.Range(func(idx int, elem *Value) {
		if !equal(c2.At(idx), elem) {
			t.Fatal("expected element not found in c2", elem, c1, c2)
		}
	})
	c2.Range(func(idx int, elem *Value) {
		if !equal(c1.At(idx), elem) {
			t.Fatal("expected element not found in c1", elem, c1, c2)
		}
	})
}

func TestArrayString(t *testing.T) {
	arr := ArrayWith(1, 2, 3, 4, 5, 6)
	if arr.String() != "[1,2,3,4,5,6]" {
		t.Fatal("array.String() didn't yeild correct result")
	}
}

func TestArrayFind(t *testing.T) {
	arr := ArrayWith(1, 2, 3, 4, 5, 6)
	t.Run("inbounds", func(t *testing.T) {
		v, ok := arr.Find(2)
		if !ok || v == nil {
			t.Fatal("didn't find an inbounds value")
		}
	})
	t.Run("out of bounds", func(t *testing.T) {
		v, ok := arr.Find(-1)
		if ok || v != nil {
			t.Fatal("found an out of bounds value")
		}
	})
}

func TestArraySort(t *testing.T) {
	expected := ArrayWith(1, 2, 3, 4, 5, 6, 7, 8)
	got := ArrayWith(8, 7, 6, 5, 4, 3, 2, 1).Sort()
	if !dyn.Equal(expected, got) {
		t.Fatalf("expected: %s\ngot: %s\n", expected, got)
	}
}

func natLess(ain, bin string) (out bool) {
	split := func(s string) []string {
		out := make([]string, 0, 3)
		var indigit bool
		var start, pos int
		var r rune
		for pos, r = range s {
			if unicode.IsDigit(r) {
				if pos > start && !indigit {
					out = append(out, s[start:pos])
					start = pos
				}
				indigit = true
			} else {
				if pos > start && indigit {
					out = append(out, s[start:pos])
					start = pos
				}
				indigit = false
			}
		}
		out = append(out, s[start:])
		return out
	}

	if ain == bin {
		return true
	}
	acomp := split(ain)
	bcomp := split(bin)
	for i, a := range acomp {
		if i >= len(bcomp) {
			return false
		}
		b := bcomp[i]
		if a == b {
			continue
		}
		if aint, err := strconv.Atoi(a); err == nil {
			if bint, err := strconv.Atoi(b); err == nil {
				return aint < bint
			}
		}
		return ain < bin
	}
	return true
}

func TestArraySortCompare(t *testing.T) {
	expected := ArrayWith("1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "20")
	got := ArrayWith("1", "10", "20", "2", "3", "6", "7", "5", "9", "8", "4").
		Sort(Compare(func(a, b *Value) int {
			if natLess(a.ToString(), b.ToString()) {
				return -1
			}
			return 1
		}))
	if !dyn.Equal(expected, got) {
		t.Fatalf("expected: %s\ngot: %s\n", expected, got)
	}
}
