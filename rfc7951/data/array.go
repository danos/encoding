// Copyright (c) 2018-2020, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"bytes"
	"reflect"
	"sort"

	"github.com/danos/encoding/rfc7951"
	"jsouthworth.net/go/immutable/vector"
)

// ArrayNew creates a new array and returns its abstract representation
func ArrayNew() *Array {
	return arrayNew()
}

func arrayNew() *Array {
	return &Array{
		store: vector.Empty(),
	}
}

// ArrayWith creates an array and initializes it with the provided elements
func ArrayWith(elements ...interface{}) *Array {
	return ArrayNew().with(elements...)
}

// ArrayFrom creates an array and initializes it with the elements from the provided slice
func ArrayFrom(in interface{}) *Array {
	return ArrayNew().from(in)
}

// Array is an RFC7159 array augmented for RFC7951 behaviors. The
// arrays are immutable, the mutation methods return new structurally
// shared copies of the original array with the changes. This provides
// cheap copies of the array and preserves the original allowing it to be
// easily shared.
type Array struct {
	store  *vector.Vector
	module string
}

// from converts a go []interface{} to an Array.
func (arr *Array) from(ins interface{}) *Array {
	val := reflect.ValueOf(ins)
	vals := make([]*Value, val.Len())
	for i := 0; i < val.Len(); i++ {
		in := val.Index(i).Interface()
		vals[i] = arr.adaptValue(ValueNew(in))
	}
	vec := vector.From(vals)
	return &Array{
		store:  vec,
		module: arr.module,
	}
}

// with returns an Array containing the elements.
func (arr *Array) with(elements ...interface{}) *Array {
	return arr.from(elements)
}

// At returns the value at the index of the array, if the index is out
// of bounds, nil is returned.
func (arr *Array) At(index int) *Value {
	if index >= arr.store.Length() || index < 0 {
		return nil
	}
	return arr.store.At(index).(*Value)
}

// Contains returns whether the index is in the bounds of the array.
func (arr *Array) Contains(index int) bool {
	return index < arr.store.Length() && index >= 0
}

// Find returns the value at the index or nil if it doesn't exist and
// whether the index was in the array.
func (arr *Array) Find(index int) (*Value, bool) {
	v, ok := arr.store.Find(index)
	if !ok {
		return nil, ok
	}
	return v.(*Value), ok
}

// Assoc associates the value with the index in the array. If the
// index is out of bounds the array is padded to that index and the value
// is associated.
func (arr *Array) Assoc(index int, value interface{}) *Array {
	newStore := arr.store
	if arr.Length() <= index {
		for i := arr.Length(); i < index+1; i++ {
			newStore = newStore.Append(nil)
		}
	}
	newStore = newStore.Assoc(index, arr.adaptValue(ValueNew(value)))
	return &Array{
		store:  newStore,
		module: arr.module,
	}
}

// Length returns the number of elements in the array.
func (arr *Array) Length() int {
	return arr.store.Length()
}

// Append adds a new value to the end of the array.
func (arr *Array) Append(value interface{}) *Array {
	newStore := arr.store.Append(arr.adaptValue(ValueNew(value)))
	return &Array{
		store:  newStore,
		module: arr.module,
	}
}

// Delete removes an element at the supplied index from the array.
func (arr *Array) Delete(index int) *Array {
	newStore := arr.store.Delete(index)
	return &Array{
		store:  newStore,
		module: arr.module,
	}
}

func (arr *Array) detect(fn func(*Value) bool) *Value {
	// TODO: is there a better name for this?
	return arr.detectAndIfNone(fn, func() *Value { return nil })
}

func (arr *Array) detectAndIfNone(fn func(*Value) bool, ifNone func() *Value) *Value {
	var out *Value
	var found bool
	arr.store.Range(func(_ int, v *Value) bool {
		if fn(v) {
			out = v
			found = true
			return false
		}
		return true
	})
	if found {
		return out
	}
	return ifNone()
}

// Range iterates over the object's members. Range can take a set of functions
// matched by type. If the function returns a bool this is treated as a
// loop terminataion variable if false the loop will terminate.
//
//     func(int, *Value) iterates over indicies and values.
//     func(int, *Value) bool
//     func(int) iterates over only the indicies
//     func(int) bool
//     func(*Value) iterates over only the values
//     func(*Value bool
func (arr *Array) Range(fn interface{}) *Array {
	switch f := fn.(type) {
	case func(int, *Value):
	case func(int, *Value) bool:
	case func(*Value):
		fn = func(idx int, val interface{}) bool {
			f(val.(*Value))
			return true
		}
	case func(*Value) bool:
		fn = func(idx int, val interface{}) bool {
			return f(val.(*Value))
		}
	case func(int):
		fn = func(idx int, val interface{}) bool {
			f(idx)
			return true
		}
	case func(int) bool:
		fn = func(idx int, val interface{}) bool {
			return f(idx)
		}
	default:
		panic("invalid range function")
	}
	arr.store.Range(fn)
	return arr
}

func (arr *Array) selectItems(fn func(*Value) bool) *Array {
	out := ArrayNew()
	out.module = arr.module
	out.store = out.store.Transform(
		func(store *vector.TVector) *vector.TVector {
			arr.Range(func(elem *Value) {
				if fn(elem) {
					elem = out.adaptValue(elem)
					store = store.Append(elem)
				}
			})
			return store
		})
	return out
}

// toNative returns a go native []interface{} from the object.
func (arr *Array) toNative() interface{} {
	out := make([]interface{}, arr.Length())
	arr.Range(func(idx int, value *Value) {
		out[idx] = value.ToNative()
	})
	return out
}

// toData returns the contents of the array as a []*Value that
// can be used with things like text/template more easily.
func (arr *Array) toData() interface{} {
	out := make([]*Value, arr.Length())
	arr.Range(func(idx int, value *Value) {
		out[idx] = value
	})
	return out
}

func (arr *Array) belongsTo(orig *Value, moduleName string) *Value {
	if moduleName == arr.module {
		return orig
	}
	out := arr.copy()
	out.module = moduleName
	out.store = out.store.Transform(
		func(store *vector.TVector) *vector.TVector {
			arr.Range(func(idx int, val *Value) {
				adaptedValue := out.adaptValue(val)
				if adaptedValue.equal(val) {
					return
				}
				store = store.Assoc(idx, adaptedValue)
			})
			return store
		})
	return ValueNew(out)
}

func (arr *Array) adaptValue(val *Value) *Value {
	return val.belongsTo(val, arr.module)
}

func (arr *Array) copy() *Array {
	return &Array{
		module: arr.module,
		store:  arr.store,
	}
}

// merge merges one array with another. The returned array is the
// old array with any existing indicies replaced with counterparts from the
// new object and any new indicies added. Merge is accretive only and will
// not remove non-existant indicies.
func (arr *Array) merge(new *Value) *Value {
	return new.Perform(func(n *Array) *Value {
		out := arr.Transform(func(out *TArray) {
			arr.Range(func(i int, v *Value) {
				if n.Contains(i) {
					out = out.Assoc(i,
						v.Merge(n.At(i)))
				}
			})
			n.Range(func(i int, v *Value) {
				if !arr.Contains(i) {
					out = out.Append(v)
				}
			})
		})
		return ValueNew(out)
	}, func(_ interface{}) *Value {
		// By default just return the original array; can't merge
		// unlike types.
		return ValueNew(arr)
	}).(*Value)
}

// Equal implements equality for arrays. An array is equal to another
// array if all their values at each index is equal. Equality checks are linear
// with respect to the number of elements.
func (arr *Array) Equal(other interface{}) bool {
	oa, isArray := other.(*Array)
	return isArray &&
		oa.module == arr.module &&
		oa.store.Length() == arr.store.Length() &&
		equal(oa.store, arr.store)
}

// String returns a string representation of the Array.
func (arr *Array) String() string {
	var buf bytes.Buffer
	arr.marshalRFC7951(&buf, arr.module)
	return buf.String()
}

func (arr *Array) marshalRFC7951(buf *bytes.Buffer, module string) error {
	buf.WriteByte('[')
	arr.Range(func(i int, v *Value) {
		v.marshalRFC7951(buf, module)
		if i < arr.Length()-1 {
			buf.WriteByte(',')
		}
	})
	buf.WriteByte(']')
	return nil
}

func (arr *Array) unmarshalRFC7951(
	msg []byte, module string,
	strs *stringInterner,
	vals *valueInterner,
) error {
	var a []rfc7951.RawMessage
	rfc7951.Unmarshal(msg, &a)
	arr.module = module
	arr.store = arr.store.Transform(
		func(store *vector.TVector) *vector.TVector {
			for _, v := range a {
				val := valueNew(nil)
				val.unmarshalRFC7951(v, arr.module, strs, vals)
				val = arr.adaptValue(val)
				val = vals.Intern(val)
				store = store.Append(val)
			}
			return store
		})
	return nil
}

func (arr *Array) diff(new *Value, path *InstanceID) []EditEntry {
	out := []EditEntry{}
	new.Perform(func(other *Array) {
		arr.Range(func(i int, v *Value) {
			if other.Contains(i) {
				out = append(out,
					v.diff(other.At(i),
						path.addPosPredicate(i))...)
			} else {
				out = append(out,
					EditEntry{
						Action: EditDelete,
						Path:   path.addPosPredicate(i),
					})
			}
		})
		other.Range(func(i int, v *Value) {
			if arr.Contains(i) {
				return
			}
			out = append(out,
				EditEntry{
					Action: EditAssoc,
					Path:   path.addPosPredicate(i),
					Value:  v,
				})
		})
	}, func(other interface{}) {
		out = []EditEntry{
			{Action: EditAssoc, Path: path, Value: ValueNew(new)},
		}
	})
	return out
}

// Transform executes the provided function against a mutable
// transient array to provide a faster, less memory intensive, array
// editing mechanism.
func (arr *Array) Transform(fn func(*TArray)) *Array {
	tarr := &TArray{
		orig:  arr,
		store: arr.store.AsTransient(),
	}
	fn(tarr)
	out := arr.copy()
	out.store = tarr.store.AsPersistent()
	return out
}

// Sort sorts an array returning a new array that is sorted.
// by default sort will use dyn.Compare as the comparison operator
// this may be overridden using the Compare option.
func (arr *Array) Sort(options ...SortOption) *Array {
	var opts sortOpts
	opts.compare = func(v1, v2 *Value) int {
		return v1.Compare(v2)
	}
	for _, opt := range options {
		opt(&opts)
	}
	out := arr.copy()
	sorter := arraySorter{
		array: out.store.AsTransient(),
		opts:  &opts,
	}
	sort.Sort(&sorter)
	out.store = sorter.array.AsPersistent()
	return out
}

type arraySorter struct {
	array *vector.TVector
	opts  *sortOpts
}

func (s *arraySorter) Len() int {
	return s.array.Length()
}

func (s *arraySorter) Less(i, j int) bool {
	return s.opts.compare(s.array.At(i).(*Value),
		s.array.At(j).(*Value)) < 0
}

func (s *arraySorter) Swap(i, j int) {
	a, b := s.array.At(i), s.array.At(j)
	s.array.Assoc(i, b)
	s.array.Assoc(j, a)
}

type sortOpts struct {
	compare func(v1, v2 *Value) int
}

// SortOption is an option to the Array.Sort function
type SortOption func(*sortOpts)

// Compare takes a comparison function and returns a sort option
// A compare function takes two values and returns a trinary state as
// an integer. Less than zero indicates the first was less than the last,
// zero indicates the two values were equal, and greater than zero
// indicates that the first was greater than the last.
func Compare(fn func(a, b *Value) int) SortOption {
	return func(opts *sortOpts) {
		opts.compare = fn
	}
}

// TArray is a transient array that may be used to perform
// transformations on an array in a fast mutable fashion. This can
// only be accessed via the (*Array).Transform method. Care should be
// taken not to share this among threads as its values are mutable.
type TArray struct {
	orig  *Array
	store *vector.TVector
}

// Assoc associates the value with the index in the array. If the
// index is out of bounds the array is padded to that index and the
// value is associated.
func (arr *TArray) Assoc(i int, v interface{}) *TArray {
	arr.store = arr.store.Assoc(i, arr.orig.adaptValue(ValueNew(v)))
	return arr
}

// Append adds a new value to the end of the array.
func (arr *TArray) Append(value interface{}) *TArray {
	arr.store = arr.store.Append(arr.orig.adaptValue(ValueNew(value)))
	return arr
}

// At returns the value at the index of the array, if the index is out
// of bounds, nil is returned.
func (arr *TArray) At(index int) *Value {
	if index >= arr.store.Length() || index < 0 {
		return nil
	}
	return arr.store.At(index).(*Value)
}

// Contains returns whether the index is in the bounds of the array.
func (arr *TArray) Contains(index int) bool {
	return index < arr.store.Length() && index >= 0
}

// Delete removes an element at the supplied index from the array.
func (arr *TArray) Delete(index int) *TArray {
	arr.store = arr.store.Delete(index)
	return arr
}

// Find returns the value at the index or nil if it doesn't exist and
// whether the index was in the array.
func (arr *TArray) Find(index int) (*Value, bool) {
	v, ok := arr.store.Find(index)
	if !ok {
		return nil, ok
	}
	return v.(*Value), ok
}

// Length returns the number of elements in the array.
func (arr *TArray) Length() int {
	return arr.store.Length()
}

// Range iterates over the object's members. Range can take a set of functions
// matched by type. If the function returns a bool this is treated as a
// loop terminataion variable if false the loop will terminate.
//
//     func(int, *Value) iterates over indicies and values.
//     func(int, *Value) bool
//     func(int) iterates over only the indicies
//     func(int) bool
//     func(*Value) iterates over only the values
//     func(*Value bool
func (arr *TArray) Range(fn interface{}) {
	// NOTE: this must be done inline to avoid needing a heap
	// allocation for the generated closure.
	switch f := fn.(type) {
	case func(int, *Value):
	case func(int, *Value) bool:
	case func(*Value):
		fn = func(idx int, val interface{}) bool {
			f(val.(*Value))
			return true
		}
	case func(*Value) bool:
		fn = func(idx int, val interface{}) bool {
			return f(val.(*Value))
		}
	case func(int):
		fn = func(idx int, val interface{}) bool {
			f(idx)
			return true
		}
	case func(int) bool:
		fn = func(idx int, val interface{}) bool {
			return f(idx)
		}
	default:
		panic("invalid range function")
	}
	arr.store.Range(fn)
}

// Sort sorts an array returning a new array that is sorted.
// by default sort will use dyn.Compare as the comparison operator
// this may be overridden using the Compare option.
func (arr *TArray) Sort(options ...SortOption) *TArray {
	var opts sortOpts
	opts.compare = func(v1, v2 *Value) int {
		return v1.Compare(v2)
	}
	for _, opt := range options {
		opt(&opts)
	}
	sorter := arraySorter{
		array: arr.store,
		opts:  &opts,
	}
	sort.Sort(&sorter)
	arr.store = sorter.array
	return arr
}

// String returns a string representation of the Array.
func (arr *TArray) String() string {
	var buf bytes.Buffer
	arr.marshalRFC7951(&buf, arr.orig.module)
	return buf.String()
}

func (arr *TArray) marshalRFC7951(buf *bytes.Buffer, module string) error {
	buf.WriteByte('[')
	arr.Range(func(i int, v *Value) {
		v.marshalRFC7951(buf, module)
		if i < arr.Length()-1 {
			buf.WriteByte(',')
		}
	})
	buf.WriteByte(']')
	return nil
}
