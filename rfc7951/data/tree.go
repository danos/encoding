// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"bytes"

	"jsouthworth.net/go/immutable/vector"
)

// TreeNew creates a new empty tree
func TreeNew() *Tree {
	return TreeFromObject(ObjectNew())
}

// TreeFromObject creates a tree rooted at the supplied object.
func TreeFromObject(obj *Object) *Tree {
	return &Tree{
		root: ValueNew(obj),
	}
}

// TreeFromValue creates a tree with a single member, 'rfc7951:data', in its
// root pointing to the supplied value.
func TreeFromValue(v *Value) *Tree {
	return TreeFromObject(ObjectWith(PairNew("rfc7951:data", v)))
}

// Tree represents an RFC7951 tree, it is rooted at an object and
// provides additional functionallity on top of the object
// functionallity. Trees are indexed using instance-identifiers
// instead of single keys. Trees are immutable and any mutation
// operation will return a new structurally shared copy of the tree with
// the changes made. This allows for cheap copies of the tree and for it
// to be shared easily.
type Tree struct {
	root *Value
}

// Root returns the tree's root Object as a Value.
func (t *Tree) Root() *Value {
	return t.root
}

// Merge merges two trees together by recursively calling Merge on the roots.
func (t *Tree) Merge(new *Tree) *Tree {
	return TreeFromObject(t.Root().
		Merge(new.Root()).
		AsObject())
}

// At returns the Value at the instance-idenfitifer provided.
func (t *Tree) At(instanceID string) *Value {
	return t.at(InstanceIDNew(instanceID))
}

func (t *Tree) at(id *InstanceID) *Value {
	return id.MatchAgainst(t.Root())
}

// Find returns the Value at the instance-identifier or nil if none,
// and whether the value is in the tree.
func (t *Tree) Find(instanceID string) (*Value, bool) {
	return t.find(InstanceIDNew(instanceID))
}

func (t *Tree) find(id *InstanceID) (*Value, bool) {
	return id.Find(t.Root())
}

// Assoc associates the value provided at the location pointed to
// by the instance-identifier.
func (t *Tree) Assoc(instanceID string, value interface{}) *Tree {
	return t.assoc(InstanceIDNew(instanceID), ValueNew(value))
}

func (t *Tree) assoc(i *InstanceID, v *Value) *Tree {
	type valueSelector struct {
		value    *Value
		selector instanceIDSelector
	}

	// Generate the operations that need to occur. This traverses
	// the InstanceID and ensures that the required nodes are created
	// for the process phase.
	queue := vector.Empty().AsTransient() // Cheap appends
	path, selector := i.path(), i.selector()
	for path != nil {
		value := path.MatchAgainst(t.Root())
		if c, isCreator := selector.(nodeCreator); isCreator &&
			value == nil {
			value = c.createNode()
		}
		queue.Append(valueSelector{
			value:    value,
			selector: selector,
		})
		path, selector = path.path(), path.selector()
	}

	// Perform the operations, this builds the new object
	// bottom up.
	queue.Range(func(_ int, vs valueSelector) {
		mm, isMatchModifier := vs.selector.(matchModifier)
		if isMatchModifier {
			v = mm.modifyMatchCriteria(v)
		}
		id := vs.selector.computeIdentifierDefault(vs.value)
		v = vs.value.Perform(
			func(o *Object) *Value {
				return ValueNew(o.Assoc(id.(string), v))
			},
			func(a *Array) *Value {
				return ValueNew(a.Assoc(id.(int), v))
			},
		).(*Value)
	})

	return TreeFromObject(v.AsObject())
}

// Delete removes the instance-identifier from the tree.
func (t *Tree) Delete(instanceID string) *Tree {
	return t.delete(InstanceIDNew(instanceID))
}

func (t *Tree) delete(i *InstanceID) *Tree {
	_, found := i.Find(t.Root())
	if !found {
		return t
	}
	path, selector := i.path(), i.selector()
	v := path.MatchAgainst(t.Root())
	id := selector.computeIdentifier(v)
	v = v.Perform(
		func(o *Object) *Value {
			return ValueNew(o.Delete(id.(string)))
		},
		func(a *Array) *Value {
			return ValueNew(a.Delete(id.(int)))
		},
	).(*Value)
	// We deleted the requested item, now we need to assoc the
	// new parent at the right location in the tree. Note for
	// schema less operations such as this one, no pruning of empty
	// lists or objects is done. That will be handled by a different
	// operation.
	return t.assoc(path, v)
}

// Contains returns whether the instance-identifer points to a node in the tree.
func (t *Tree) Contains(instanceID string) bool {
	_, found := InstanceIDNew(instanceID).
		Find(t.Root())
	return found
}

// Length returns the number of elements in the tree.
func (t *Tree) Length() int {
	var count int
	t.Range(func(*Value) {
		count++
	})
	return count
}

// Range iterates over the Trees's paths. Range can take a set of functions
// matched by type. If the function returns a bool this is treated as a
// loop terminataion variable if false the loop will terminate.
//
//     func(*InstanceID, *Value) iterates over paths as an instance-identifier
//                               and values.
//     func(*InstanceID, *Value) bool
//     func(string, *Value) iterates over paths and values.
//     func(string, *Value) bool
//     func(*InstanceID) iterates over only the paths as an instance-identifier
//     func(*InstanceID) bool
//     func(string) iterates over only the paths
//     func(string) bool
//     func(*Value) iterates over only the values
//     func(*Value) bool
func (t *Tree) Range(fn interface{}) *Tree {
	iid := &InstanceID{}
	rangeFn := genTreeRangeFunc(fn)
	var recur func(*InstanceID, *Value) bool
	recur = func(iid *InstanceID, elem *Value) bool {
		return elem.Perform(func(o *Object) bool {
			var cont bool
			cont = rangeFn(iid, ValueNew(o))
			if !cont {
				return false
			}
			o.Range(func(key string, v *Value) bool {
				cont = recur(iid.push(key), v)
				return cont
			})
			return cont
		}, func(a *Array) bool {
			var cont bool
			cont = rangeFn(iid, ValueNew(a))
			if !cont {
				return false
			}
			a.Range(func(i int, v *Value) bool {
				cont = recur(iid.addPosPredicate(i), v)
				return cont
			})
			return cont

		}, func(other *Value) bool {
			return rangeFn(iid, other)
		}).(bool)
	}
	t.root.AsObject().
		Range(func(key string, v *Value) bool {
			return recur(iid.push(key), v)
		})
	return t
}

func genTreeRangeFunc(fn interface{}) func(iid *InstanceID, v *Value) bool {
	switch f := fn.(type) {
	case func(*InstanceID, *Value) bool:
		return f
	case func(*InstanceID, *Value):
		return func(iid *InstanceID, value *Value) bool {
			f(iid, value)
			return true
		}
	case func(string, *Value) bool:
		return func(iid *InstanceID, value *Value) bool {
			return f(iid.String(), value)
		}
	case func(string, *Value):
		return func(iid *InstanceID, value *Value) bool {
			f(iid.String(), value)
			return true
		}
	case func(*Value) bool:
		return func(_ *InstanceID, value *Value) bool {
			return f(value)
		}
	case func(*Value):
		return func(_ *InstanceID, value *Value) bool {
			f(value)
			return true
		}
	case func(*InstanceID) bool:
		return func(iid *InstanceID, _ *Value) bool {
			return f(iid)
		}
	case func(*InstanceID):
		return func(iid *InstanceID, _ *Value) bool {
			f(iid)
			return true
		}
	case func(string) bool:
		return func(iid *InstanceID, _ *Value) bool {
			return f(iid.String())
		}
	case func(string):
		return func(iid *InstanceID, _ *Value) bool {
			f(iid.String())
			return true
		}
	default:
		panic("invalid range function")
	}
}

// MarshalRFC7951 returns the Tree encoded as RFC7951 data.
func (t *Tree) MarshalRFC7951() ([]byte, error) {
	var buf bytes.Buffer
	err := t.Root().marshalRFC7951(&buf, "")
	return buf.Bytes(), err
}

// UnmarshalRFC7951 fills out the Tree from the RFC7951 encoded
// message. This can't be fully immutable, the caller has to ensure
// the array isn't used until unmarshal is finished.
func (t *Tree) UnmarshalRFC7951(msg []byte) error {
	if t.root == nil {
		t.root = ValueNew(ObjectNew())
	}
	return t.root.UnmarshalRFC7951(msg)
}

// Equal implements equality for the tree. It compares the roots for
// equality.
func (t *Tree) Equal(other interface{}) bool {
	ot, isTree := other.(*Tree)
	if !isTree {
		return false
	}
	return equal(t.Root(), ot.Root())
}

// String returns a string representation of the tree.
func (t *Tree) String() string {
	return t.Root().String()
}

// Diff compares two trees and returns the operations required to edit
// the original to produce the other one.
func (t *Tree) Diff(other *Tree) *EditOperation {
	return &EditOperation{
		Actions: t.Root().diff(other.Root(), &InstanceID{}),
	}
}

// Edit applies an EditOperation to the tree. This allows for capturing large
// change sets as a piece of data than can be evaluated as tree operations
// and applied to the tree.
func (t *Tree) Edit(edit *EditOperation) *Tree {
	op := edit.eval()
	return op(t)
}
