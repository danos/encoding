// Copyright (c) 2019, AT&T Intellectual Property.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"errors"
	"fmt"

	"github.com/danos/encoding/rfc7951"
)

const (
	// EditAssoc is the edit action association with the Assoc operation.
	EditAssoc EditAction = "assoc"
	// EditDelete is the edit action association with the Delete operation.
	EditDelete EditAction = "delete"
	// EditMerge is the edit action association with the Merge operation.
	EditMerge EditAction = "merge"
)

// EditAction is an action that can be performed by the edit engine.
type EditAction string

// UnmarshalRFC7951 unmarshals the RFC7951 encoded message into the EditAction.
func (e *EditAction) UnmarshalRFC7951(msg []byte) error {
	var s string
	err := rfc7951.Unmarshal(msg, &s)
	if err != nil {
		return err
	}
	switch s {
	case "assoc":
		*e = EditAssoc
	case "delete":
		*e = EditDelete
	case "merge":
		*e = EditMerge
	default:
		return errors.New("unknown edit-action" + string(msg))
	}
	return nil
}

// MarshalRFC7951 returns the EditAction as RFC7951 encoded data.
func (e EditAction) MarshalRFC7951() ([]byte, error) {
	switch e {
	case EditAssoc, EditDelete, EditMerge:
		s := e.String()
		return []byte("\"" + s + "\""), nil
	default:
		return nil, fmt.Errorf("unknown edit-action %v", e)
	}
}

// String returns the EditAction as a string.
func (e EditAction) String() string {
	return string(e)
}

// EditEntry contains the actions to perform as well as the
// instance-id to perform it at and the value if any to be used.
type EditEntry struct {
	Action EditAction  `rfc7951:"action"`
	Path   *InstanceID `rfc7951:"path"`
	Value  *Value      `rfc7951:"value,omitempty"`
}

func (e *EditEntry) evalAssoc() func(*Tree) *Tree {
	path, value := e.Path, e.Value
	return func(t *Tree) *Tree {
		return t.assoc(path, value)
	}
}
func (e *EditEntry) evalDelete() func(*Tree) *Tree {
	path := e.Path
	return func(t *Tree) *Tree {
		return t.delete(path)
	}
}
func (e *EditEntry) evalMerge() func(*Tree) *Tree {
	path, value := e.Path, e.Value
	return func(t *Tree) *Tree {
		val := t.at(path)
		val = val.Merge(value)
		return t.assoc(path, val)
	}
}
func (e *EditEntry) eval() func(*Tree) *Tree {
	switch e.Action {
	case EditAssoc:
		return e.evalAssoc()
	case EditDelete:
		return e.evalDelete()
	case EditMerge:
		return e.evalMerge()
	default:
		panic(fmt.Errorf("unknown edit-action %v", e.Action))
	}
}

// EditOperation holds edit actions and allow them to
// be encoded as RFC7951 data.
type EditOperation struct {
	Actions []EditEntry `rfc7951:"actions,omitempty"`
}

// String returns a string representation of the EditOperation.
func (e *EditOperation) String() string {
	data, _ := rfc7951.Marshal(e)
	return string(data)
}

func (e *EditOperation) eval() func(*Tree) *Tree {
	actions := make([]func(*Tree) *Tree, len(e.Actions))
	for i, action := range e.Actions {
		actions[i] = action.eval()
	}
	return func(t *Tree) *Tree {
		for _, action := range actions {
			t = action(t)
		}
		return t
	}
}

// EditOperationNew produces a new EditOperation from the
// provided entries. This allows one to declaratively build an
// EditOperation.
func EditOperationNew(entries ...EditEntry) *EditOperation {
	return &EditOperation{
		Actions: entries,
	}
}

type editEntryOptions struct {
	value *Value
}

// EditEntryOption is a constructor for the optional parts of an EditEntry.
type EditEntryOption func(*editEntryOptions)

// EditEntryValue produce an EditEntryOption that populates the value field
// of an EditEntry.
func EditEntryValue(val interface{}) EditEntryOption {
	return func(o *editEntryOptions) {
		o.value = ValueNew(val)
	}
}

// EditEntryNew constructs a new EditEntry from the provided parameters.
// The last option in wins if they write the same option.
func EditEntryNew(action EditAction, path string, options ...EditEntryOption) EditEntry {
	var opts editEntryOptions
	for _, option := range options {
		option(&opts)
	}
	return EditEntry{
		Action: action,
		Path:   InstanceIDNew(path),
		Value:  opts.value,
	}
}
