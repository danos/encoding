// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/danos/encoding/rfc7951"
	"jsouthworth.net/go/immutable/hashmap"
)

// ObjectNew creates a new object.
func ObjectNew() *Object {
	return objectNew()
}

func objectNew() *Object {
	return &Object{
		store: hashmap.Empty(),
	}
}

// ObjectWith creates a new object and then populates it with the supplied pairs
func ObjectWith(pairs ...Pair) *Object {
	return ObjectNew().with(pairs...)
}

// ObjectFrom creates a new object and then populates it with the data from the supplied map
func ObjectFrom(in map[string]interface{}) *Object {
	return ObjectNew().from(in)
}

// PairNew creates a new pair
func PairNew(key string, value interface{}) Pair {
	return Pair{key: key, value: ValueNew(value)}
}

// Pair is a key/value pair. These are representations of the members
// of Objects per RFC7159.
type Pair struct {
	key   string
	value *Value
}

// Key returns the key.
func (p Pair) Key() string { return p.key }

// Value returns the value.
func (p Pair) Value() *Value { return p.value }

// String returns a string representation of the Pair.
func (p Pair) String() string { return fmt.Sprintf("[%v %v]", p.key, p.value) }

// Equal implements equality between Pairs.
func (p Pair) Equal(other interface{}) bool {
	op, isPair := other.(Pair)
	if !isPair {
		return false
	}
	return op.key == p.key && equal(op.value, p.value)
}

// Object is an RFC7159 (JSON) object augmented for RFC7951
// behaviors. These objects are immutable, the mutation methods return a
// structurally shared copy of the object with the required
// changes. This provides cheap copies of the object and preserves
// the original allowing it to be easily shared. Objects store the
// module:key as the full key, but one may access operations using only
// the key if the module is the same as the parent object.
type Object struct {
	store  *hashmap.Map
	module string
}

// from converts a native go map to an Object.
func (obj *Object) from(in map[string]interface{}) *Object {
	out := obj.copy()
	out.store = out.store.Transform(
		func(store *hashmap.TMap) *hashmap.TMap {
			for k, v := range in {
				key, val := obj.adaptValue(k, ValueNew(v))
				store = store.Assoc(key, val)
			}
			return store
		})
	return out
}

// with allows one to build an object from a list of Pairs. This provides
// a declarative mechanism for producing an object.
func (obj *Object) with(pairs ...Pair) *Object {
	out := obj.copy()
	out.store = out.store.Transform(
		func(store *hashmap.TMap) *hashmap.TMap {
			for _, pair := range pairs {
				k, v := obj.adaptValue(pair.Key(), pair.Value())
				store = store.Assoc(k, v)
			}
			return store
		})
	return out
}

// Range iterates over the object's members. Range can take a set of functions
// matched by type. If the function returns a bool this is treated as a
// loop terminataion variable if false the loop will terminate.
//
//     func(Pair) iterates over Pairs
//     func(Pair) bool, called with a Pair, terminates the loop on false.
//     func(string, *Value) iterates over keys and values.
//     func(string, *Value) bool
//     func(string) iterates over only the keys
//     func(string) bool
//     func(*Value) iterates over only the values
//     func(*Value bool
func (obj *Object) Range(fn interface{}) *Object {
	switch f := fn.(type) {
	case func(Pair):
		fn = func(e hashmap.Entry) bool {
			f(PairNew(e.Key().(string), e.Value()))
			return true
		}
	case func(Pair) bool:
		fn = func(e hashmap.Entry) bool {
			return f(PairNew(e.Key().(string), e.Value()))
		}
	case func(string, *Value):
		fn = func(e hashmap.Entry) bool {
			f(e.Key().(string), e.Value().(*Value))
			return true
		}
	case func(string, *Value) bool:
		fn = func(e hashmap.Entry) bool {
			return f(e.Key().(string), e.Value().(*Value))
		}
	case func(*Value):
		fn = func(e hashmap.Entry) bool {
			f(e.Value().(*Value))
			return true
		}
	case func(*Value) bool:
		fn = func(e hashmap.Entry) bool {
			return f(e.Value().(*Value))
		}
	case func(string):
		fn = func(e hashmap.Entry) bool {
			f(e.Key().(string))
			return true
		}
	case func(string) bool:
		fn = func(e hashmap.Entry) bool {
			return f(e.Key().(string))
		}
	default:
		panic("invalid range function")
	}
	obj.store.Range(fn)
	return obj
}

// At returns the Value at the key's location or nil if it doesn't exist.
// The key may be either 'module:key' or just key if the module is the same
// as the containing object's module.
func (obj *Object) At(key string) *Value {
	k := obj.adaptKey(key)
	out, ok := obj.store.Find(k)
	if !ok {
		return nil
	}
	return out.(*Value)
}

// Contains returns true if the key exists in the object.
// The key may be either 'module:key' or just key if the module is the same
// as the containing object's module.
func (obj *Object) Contains(key string) bool {
	k := obj.adaptKey(key)
	return obj.store.Contains(k)
}

// Find returns the value at the key or nil if it doesn't exist and
// whether the key was in the object.
func (obj *Object) Find(key string) (*Value, bool) {
	k := obj.adaptKey(key)
	out, ok := obj.store.Find(k)
	if !ok {
		return nil, ok
	}
	return out.(*Value), ok
}

// Assoc associates a new value with the key.
// The key may be either 'module:key' or just key if the module is the same
// as the containing object's module.
func (obj *Object) Assoc(key string, value interface{}) *Object {
	k, v := obj.adaptValue(key, ValueNew(value))
	new := obj.store.Assoc(k, v)
	if new == obj.store {
		return obj
	}
	return &Object{
		store:  new,
		module: obj.module,
	}
}

// Length returns the number of elements in the object.
func (obj *Object) Length() int {
	return obj.store.Length()
}

// Delete removes a key from the object.
// The key may be either 'module:key' or just key if the module is the same
// as the containing object's module.
func (obj *Object) Delete(key string) *Object {
	k := obj.adaptKey(key)
	new := obj.store.Delete(k)
	if new == obj.store {
		return obj
	}
	return &Object{
		store:  new,
		module: obj.module,
	}
}

// toNative produces a go native map[string]interface{} from the object.
func (obj *Object) toNative() interface{} {
	out := make(map[string]interface{})
	obj.Range(func(assoc Pair) {
		out[assoc.Key()] = assoc.Value().ToNative()
	})
	return out
}

// toData returns the contents of an object as a map[string]*Value that
// can be used with things like text/template more easily.
func (obj *Object) toData() interface{} {
	out := make(map[string]*Value)
	obj.Range(func(key string, val *Value) {
		out[key] = val
	})
	return out
}

func (obj *Object) adaptValue(k string, val *Value) (string, *Value) {
	module, _ := obj.parseKey(k)
	val = val.belongsTo(module)
	key := obj.adaptKey(k)
	return key, val
}

func (obj *Object) belongsTo(moduleName string) *Value {
	if moduleName == obj.module {
		return ValueNew(obj)
	}
	oldModule := obj.module
	new := obj.copy()
	new.module = moduleName
	new.store = new.store.Transform(
		func(newStore *hashmap.TMap) *hashmap.TMap {
			obj.Range(func(key string, val *Value) {
				module, _ := obj.parseKey(key)
				switch module {
				case "", oldModule:
					k, v := new.adaptValue(key, val)
					newStore.Assoc(k, v)
					newStore.Delete(obj.adaptKey(key))
				default:
					return
				}
			})
			return newStore
		})
	return ValueNew(new)
}

func (obj *Object) adaptKey(key string) string {
	module, key := obj.parseKey(key)
	if module == "" {
		return key
	}
	return module + ":" + key
}

func (obj *Object) parseKey(k string) (string, string) {
	elems := strings.SplitN(k, ":", 2)
	switch len(elems) {
	case 1:
		return obj.module, elems[0]
	default:
		return elems[0], elems[1]
	}
}

func (obj *Object) copy() *Object {
	return &Object{
		module: obj.module,
		store:  obj.store,
	}

}

// merge merges one object with another. The returned object is the
// old object with any existing keys replaced with counterparts from the
// new object and any new keys added. Merge is accretive only and will
// not remove non-existant keys.
func (obj *Object) merge(new *Value) *Value {
	return new.Perform(func(n *Object) *Value {
		out := obj.copy()
		out.store = out.store.Transform(
			func(store *hashmap.TMap) *hashmap.TMap {
				obj.Range(func(key string, val *Value) {
					k, v := out.adaptValue(key, val)
					if n.Contains(k) {
						store = store.Assoc(k,
							v.Merge(n.At(k)))
					}
				})
				n.Range(func(key string, val *Value) {
					k, v := out.adaptValue(key, val)
					if !store.Contains(k) {
						store = store.Assoc(k, v)
					}
				})
				return store
			})
		return ValueNew(out)
	}, func(_ interface{}) *Value {
		// By default just return the original object; can't merge
		// unlike types.
		return ValueNew(obj)
	}).(*Value)
}

// Equal implements equality for objects. An object is equal to another
// object if all their keys contains equal values. Equality checks are linear
// with respect to the number of keys.
func (obj *Object) Equal(other interface{}) bool {
	oo, isObject := other.(*Object)
	return isObject &&
		oo.module == obj.module &&
		oo.store.Length() == obj.store.Length() &&
		equal(oo.store, obj.store)
}

// String returns a string representation of the Object.
func (obj *Object) String() string {
	var buf bytes.Buffer
	obj.marshalRFC7951(&buf, obj.module)
	return buf.String()
}

func (obj *Object) marshalRFC7951(buf *bytes.Buffer, module string) error {
	buf.WriteByte('{')
	var n int
	obj.Range(func(pair Pair) {
		k := pair.Key()
		mod, key := obj.parseKey(k)
		if mod == module {
			k = key
		}
		buf.WriteByte('"')
		buf.WriteString(k)
		buf.WriteByte('"')
		buf.WriteByte(':')
		pair.Value().marshalRFC7951(buf, mod)
		if n < obj.Length()-1 {
			buf.WriteByte(',')
		}
		n = n + 1
	})
	buf.WriteByte('}')
	return nil
}

func (obj *Object) unmarshalRFC7951(msg []byte, module string) error {
	// This can't be fully immutable, the caller has to ensure
	// the object isn't used until unmarshal is finished, this
	// shouldn't be a problem in practice...
	var m map[string]rfc7951.RawMessage
	rfc7951.Unmarshal(msg, &m)
	obj.module = module
	obj.store = obj.store.Transform(
		func(store *hashmap.TMap) *hashmap.TMap {
			for k, v := range m {
				val := valueNew(nil)
				module, _ := obj.parseKey(k)
				val.unmarshalRFC7951(v, module)
				k, v := obj.adaptValue(k, val)
				store = store.Assoc(k, v)
			}
			return store
		})
	return nil
}

func (obj *Object) diff(new *Value, path *InstanceID) []EditEntry {
	out := []EditEntry{}
	new.Perform(func(other *Object) {
		obj.Range(func(k string, v *Value) {
			if other.Contains(k) {
				out = append(out,
					v.diff(other.At(k), path.push(k))...)
			} else {
				out = append(out,
					EditEntry{
						Action: EditDelete,
						Path:   path.push(k),
					})
			}
		})
		other.Range(func(k string, v *Value) {
			if obj.Contains(k) {
				return
			}
			out = append(out,
				EditEntry{
					Action: EditAssoc,
					Path:   path.push(k),
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
