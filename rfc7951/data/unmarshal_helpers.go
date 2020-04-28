// Copyright (c) 2020, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

type stringInterner struct {
	vals map[string]string
}

func (i *stringInterner) Intern(str string) string {
	out, ok := i.vals[str]
	if ok {
		return out
	}
	i.vals[str] = str
	return str
}

func stringInternerNew() *stringInterner {
	return &stringInterner{
		vals: make(map[string]string),
	}
}

type valueInterner struct {
	vals map[interface{}]*Value
}

func (i *valueInterner) Intern(val *Value) *Value {
	data := val.ToInterface()
	out, ok := i.vals[data]
	if ok {
		return out
	}
	i.vals[data] = val
	return val
}

func valueInternerNew() *valueInterner {
	return &valueInterner{
		vals: make(map[interface{}]*Value),
	}
}
