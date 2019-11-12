// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"testing"
)

func TestInstanceIDParsing(t *testing.T) {
	runTest := func(test, expected string) {
		t.Run(test, func(t *testing.T) {
			iid := InstanceIDNew(test).String()
			if iid != expected {
				t.Fatalf("expected %s, got %s\n", expected, iid)
			}
		})
	}
	const sQExpected = "/ietf-interfaces:interfaces/interface[name='eth0']/ietf-ip:ipv4/ip"
	sQTests := []string{
		"/ietf-interfaces:interfaces/interface[name='eth0']/ietf-ip:ipv4/ip",
		"/ietf-interfaces:interfaces/ietf-interfaces:interface[name='eth0']/ietf-ip:ipv4/ip",
		"/ietf-interfaces:interfaces/interface[	name='eth0']/ietf-ip:ipv4/ip",
		"/ietf-interfaces:interfaces/interface[name='eth0'	]/ietf-ip:ipv4/ip",
		"/ietf-interfaces:interfaces/interface[  name='eth0'	]/ietf-ip:ipv4/ip",
		"/ietf-interfaces:interfaces/interface[  name='eth0'	  ]/ietf-ip:ipv4/ip",
		"/ietf-interfaces:interfaces/ietf-interfaces:interface[name ='eth0']/ietf-ip:ipv4/ip",
		"/ietf-interfaces:interfaces/ietf-interfaces:interface[name = 'eth0']/ietf-ip:ipv4/ip",
		"/ietf-interfaces:interfaces/ietf-interfaces:interface[name	= 'eth0']/ietf-ip:ipv4/ip",
		"/ietf-interfaces:interfaces/ietf-interfaces:interface[name	= 	'eth0']/ietf-ip:ipv4/ip",
		"/ietf-interfaces:interfaces/interface[name=\"eth0\"]/ietf-ip:ipv4/ip",
	}
	for _, test := range sQTests {
		runTest(test, sQExpected)
	}
	runTest("/m:foo[id=\"bar\"]", "/m:foo[id='bar']")
	runTest("/m:foo/bar[id=\"baz\"][id2=\"quux\"]",
		"/m:foo/bar[id='baz'][id2='quux']")
	runTest("/m:foo[0]", "/m:foo[0]")
	runTest("/m:foo[.='123']", "/m:foo[.='123']")
}

func TestInstanceIDParsingFailures(t *testing.T) {
	tFunc := func(test, expFailure string) {
		t.Run(test, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					return
				}
				if err, ok := r.(error); ok {
					if err.Error() != expFailure {
						t.Fatalf("unexpected error occured: %s",
							err)
					}
					return
				}
				panic(r)
			}()
			InstanceIDNew(test)
		})
	}
	tFunc("/foo",
		"invalid instance identifier: unable to determine prefix")
	tFunc("",
		"invalid instance identifier: must specify at least one node-identifier")
	tFunc("foo",
		"invalid instance identifier: must start with a \"/\"")
	tFunc("/",
		"invalid instance identifier: must specify at least one node-identifier")
	tFunc("/foo[id='foo]",
		"invalid instance identifier: unterminated quote")
	tFunc("/xml2:m",
		"invalid instance identifier: invalid identifier, not allowed to start with xml: xml2")
	tFunc("/foo?:m",
		"invalid instance identifier: invalid node-identifier foo?")
	tFunc("/?foo:m",
		"invalid instance identifier: invalid node-identifier ?foo")
	tFunc("/m:foo[b[a='b']='c']",
		"invalid instance identifier: nested predicates are not allowed")
	tFunc("/m:foo[b='c'",
		"invalid instance identifier: unterminated predicate")
	tFunc("/m:foo[b]",
		"invalid instance identifier: invalid predicate expression b")
	tFunc("/m:foo[b=c]", "invalid instance identifier: invalid predicate, expected ''' or '\"'")
}

func TestInstanceIDMatchAgainst(t *testing.T) {
	//Test Matching semantics against an example
	obj := ObjectWith(
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
				PairNew("baz", "foo")),
			ObjectWith(
				PairNew("quux", "bar"),
				PairNew("baz", "baz")))))
	cases := []struct {
		path string
		val  interface{}
	}{
		{"/module-v1:foo/baz", "quux"},
		{"/module-v1:foo/bar/baz[0]", "quux"},
		{"/module-v1:foo/bar/baz[.='foo']", "foo"},
		{"/module-v2:baz[quux='foo']/baz", "bar"},
		{"/module-v2:baz[quux='foo'][baz='bar']/baz", "bar"},
		{"/module-v2:baz[quux='bar'][baz='baz']/baz", "baz"},
		{"/module-v2:baz[quux='foo'][baz='foo']", nil},
		{"/module-v2:baz[zuux='foo'][baz='foo']", nil},
		{"/module-v2:baz[zuux='foo'][baz='foo']/bar", nil},
		{"/module-v1:foo/nope/stillno", nil},
		{"/module-v2:baz[1]/quux", "bar"},
		{"/module-v1:foo/v2:zzz", "abc"},
		{"/module-v1:foo/v3:zzz", nil},
	}
	val := ValueNew(obj)
	for _, test := range cases {
		t.Run(test.path, func(t *testing.T) {
			t.Log("path", test.path)
			v, found := InstanceIDNew(test.path).
				Find(val)
			switch {
			case !found && test.val == nil:
				return
			case !found && test.val != nil:
				t.Fatalf("test %s expected %v, got %v",
					test.path, test.val, v)
			case v.RFC7951String() != test.val:
				t.Fatalf("test %s expected %v, got %v",
					test.path, test.val, v)
			}
		})
	}
}

func TestInstanceIDPathAndSelector(t *testing.T) {
	obj := ObjectWith(
		PairNew("module-v1:foo", ObjectWith(
			PairNew("bar", ObjectWith(
				PairNew("baz", ArrayWith("quux", "foo")),
				PairNew("quux", "quuz"))),
			PairNew("baz", "quux"),
			PairNew("v2:zzz", "abc"))),
		PairNew("module-v1:bar", "baz"),
		PairNew("module-v1:list", ArrayWith(
			ObjectWith(
				PairNew("key", "a"),
				PairNew("value", "b")),
			ObjectWith(
				PairNew("key", "c"),
				PairNew("value", "d")))),
		PairNew("module-v2:baz", ArrayWith(
			ObjectWith(
				PairNew("quux", "foo"),
				PairNew("baz", "bar")),
			ObjectWith(
				PairNew("quux", "bar"),
				PairNew("baz", "foo")),
			ObjectWith(
				PairNew("quux", "bar"),
				PairNew("baz", "baz")))))
	cases := []struct {
		path string
		id   interface{}
	}{
		{"/module-v2:baz[quux='bar'][baz='baz']", 2},   //list
		{"/module-v2:baz[quux='bar'][baz='foo']", 1},   //list
		{"/module-v2:baz[quux='foo'][baz='bar']", 0},   //list
		{"/module-v2:baz[quux='foo'][baz='baz']", nil}, //list
		{"/module-v1:list[key='a']", 0},                //list
		{"/module-v1:list[key='c']", 1},                //list
		{"/module-v1:list[key='d']", nil},              //list
		{"/module-v1:list[0]", 0},                      //list
		{"/module-v1:list[1]", 1},                      //list
		{"/module-v1:list[2]", nil},                    //list
		{"/module-v1:foo/bar/baz[.='foo']", 1},         //leaf-list
		{"/module-v1:foo/bar/baz[.='bar']", nil},       //leaf-list
		{"/module-v1:foo/bar/baz[0]", 0},               //leaf-list
		{"/module-v1:foo/bar/baz[1]", 1},               //leaf-list
		{"/module-v1:foo/bar/baz[2]", nil},             //leaf-list
		{"/module-v1:foo/bar/baz", "module-v1:baz"},    //leaf
		{"/module-v1:foo/bar/nope", nil},               //leaf
		{"/module-v1:foo/bar", "module-v1:bar"},        //container
		{"/module-v1:foo/nope", nil},                   //container
	}
	for _, test := range cases {
		t.Run(test.path, func(t *testing.T) {
			iid := InstanceIDNew(test.path)
			identifier := iid.selector().
				computeIdentifier(iid.path().
					MatchAgainst(ValueNew(obj)))
			if identifier != test.id {
				t.Fatalf("did not find the expected object, expected %v, got %v",
					test.id, identifier)
			}
		})
	}
}

var TESTOBJ = ObjectWith(
	PairNew("module-v1:leaf", "foo"),
	PairNew("module-v1:leaf-list", ArrayWith(1, 2, 3, 4, 5, 6, 7)),
	PairNew("module-v1:list", ArrayWith(
		ObjectWith(
			PairNew("key", "foo"),
			PairNew("objleaf", "bar")),
		ObjectWith(
			PairNew("key", "bar"),
			PairNew("objleaf", "baz")),
		ObjectWith(
			PairNew("key", "baz"),
			PairNew("objleaf", "quux")),
		ObjectWith(
			PairNew("key", "quux"),
			PairNew("objleaf", "quuz")))),
	PairNew("module-v1:container", ObjectWith(
		PairNew("containerleaf", "foo"))),
	PairNew("module-v1:nested", ObjectWith(
		PairNew("module-v1:leaf", "foo"),
		PairNew("module-v1:leaf-list",
			ArrayWith(1, 2, 3, 4, 5, 6, 7)),
		PairNew("module-v1:list", ArrayWith(
			ObjectWith(
				PairNew("key", "foo"),
				PairNew("objleaf", "bar")),
			ObjectWith(
				PairNew("key", "bar"),
				PairNew("objleaf", "baz")),
			ObjectWith(
				PairNew("key", "baz"),
				PairNew("objleaf", "quux")),
			ObjectWith(
				PairNew("key", "quux"),
				PairNew("objleaf", "quuz")))),
		PairNew("module-v1:container", ObjectWith(
			PairNew("containerleaf", "foo"))))),
	PairNew("module-v1:nested-list", ArrayWith(
		ObjectWith(
			PairNew("key", "nest1"),
			PairNew("module-v1:leaf", "foo"),
			PairNew("module-v1:leaf-list",
				ArrayWith(1, 2, 3, 4, 5, 6, 7)),
			PairNew("module-v1:list", ArrayWith(
				ObjectWith(
					PairNew("key", "foo"),
					PairNew("objleaf", "bar")),
				ObjectWith(
					PairNew("key", "bar"),
					PairNew("objleaf", "baz")),
				ObjectWith(
					PairNew("key", "baz"),
					PairNew("objleaf", "quux")),
				ObjectWith(
					PairNew("key", "quux"),
					PairNew("objleaf", "quuz")))),
			PairNew("module-v1:container", ObjectWith(
				PairNew("containerleaf", "foo")))),
		ObjectWith(
			PairNew("key", "nest2"),
			PairNew("module-v1:leaf", "foo"),
			PairNew("module-v1:leaf-list",
				ArrayWith(1, 2, 3, 4, 5, 6, 7)),
			PairNew("module-v1:list", ArrayWith(
				ObjectWith(
					PairNew("key", "foo"),
					PairNew("objleaf", "bar")),
				ObjectWith(
					PairNew("key", "bar"),
					PairNew("objleaf", "baz")),
				ObjectWith(
					PairNew("key", "baz"),
					PairNew("objleaf", "quux")),
				ObjectWith(
					PairNew("key", "quux"),
					PairNew("objleaf", "quuz")))),
			PairNew("module-v1:container", ObjectWith(
				PairNew("containerleaf", "foo")))))),
)
