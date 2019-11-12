// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"testing"

	"github.com/danos/encoding/rfc7951"
)

func TestTreeMerge(t *testing.T) {
	one := TreeFromObject(ObjectFrom(map[string]interface{}{
		"non-merged-v1:leaf": 1,
		"merged:leaf":        1,
		"non-merged-v1:container": map[string]interface{}{
			"foo": 1,
			"bar": 2,
		},
		"merged:container": map[string]interface{}{
			"foo":  1,
			"bar":  1,
			"quux": 1,
		},
		"merged:leaf-list": []interface{}{
			1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12,
		},
		"merged:leaf-list-longer-new": []interface{}{
			1, 2, 3, 4, 5,
		},
		"merged:list": []interface{}{
			map[string]interface{}{
				"foo":               1,
				"non-merged-v1:bar": 1,
				"quux":              1,
			},
			map[string]interface{}{
				"foo":               2,
				"non-merged-v1:bar": 2,
				"quux":              2,
			},
		},
		"merged:empty":        Empty(),
		"non-merged-v1:empty": Empty(),
		"merged:leaf-list-other-not-array": []interface{}{
			1, 2, 3, 4, 5,
		},
		"merged:container-other-not-object": map[string]interface{}{
			"foo": 1,
			"bar": 2,
		},
	}))
	two := TreeFromObject(ObjectFrom(map[string]interface{}{
		"non-merged-v2:leaf": 1,
		"merged:leaf":        2,
		"non-merged-v2:container": map[string]interface{}{
			"foo": 1,
			"bar": 2,
		},
		"merged:container": map[string]interface{}{
			"foo": 1,
			"bar": 2,
			"baz": 2,
		},
		"merged:leaf-list": []interface{}{
			1, 2, 3, 4, 10, 11, 12, 13, 14, 15,
		},
		"merged:leaf-list-longer-new": []interface{}{
			1, 2, 3, 4, 10, 11, 12, 13, 14, 15,
		},
		"merged:list": []interface{}{
			map[string]interface{}{
				"foo":               2,
				"non-merged-v2:bar": 2,
				"baz":               2,
			},
			map[string]interface{}{
				"foo":               3,
				"non-merged-v2:bar": 3,
				"baz":               3,
			},
		},
		"merged:empty":                      Empty(),
		"non-merged-v2:empty":               Empty(),
		"merged:leaf-list-other-not-array":  1,
		"merged:container-other-not-object": 1,
	}))
	expected := TreeFromObject(ObjectFrom(map[string]interface{}{
		"non-merged-v1:leaf": 1,
		"non-merged-v2:leaf": 1,
		"merged:leaf":        2,
		"non-merged-v1:container": map[string]interface{}{
			"foo": 1,
			"bar": 2,
		},
		"non-merged-v2:container": map[string]interface{}{
			"foo": 1,
			"bar": 2,
		},
		"merged:container": map[string]interface{}{
			"foo":  1,
			"bar":  2,
			"baz":  2,
			"quux": 1,
		},
		"merged:leaf-list": []interface{}{
			1, 2, 3, 4, 10, 11, 12, 13, 14, 15, 11, 12,
		},
		"merged:leaf-list-longer-new": []interface{}{
			1, 2, 3, 4, 10, 11, 12, 13, 14, 15,
		},
		"merged:list": []interface{}{
			map[string]interface{}{
				"non-merged-v1:bar": 1,
				"quux":              1,
				"foo":               2,
				"non-merged-v2:bar": 2,
				"baz":               2,
			},
			map[string]interface{}{
				"non-merged-v1:bar": 2,
				"quux":              2,
				"foo":               3,
				"non-merged-v2:bar": 3,
				"baz":               3,
			},
		},
		"merged:empty":        Empty(),
		"non-merged-v1:empty": Empty(),
		"non-merged-v2:empty": Empty(),
		"merged:leaf-list-other-not-array": []interface{}{
			1, 2, 3, 4, 5,
		},
		"merged:container-other-not-object": map[string]interface{}{
			"foo": 1,
			"bar": 2,
		},
	}))
	got := one.Merge(two)
	if !equal(got, expected) {
		t.Fatalf("Didn't get expected merge result\n\ngot:%s\n\nexpected:%s\n", got, expected)
	}
}

func TestTreeAssoc(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		value interface{}
	}{
		{
			name:  "existing leaf replacement",
			path:  "/module-v1:container/containerleaf",
			value: ValueNew("!!!"),
		},
		{
			name:  "nested leaf replacement",
			path:  "/module-v1:nested/container/containerleaf",
			value: ValueNew("!!!"),
		},
		{
			name:  "nested-list leaf replacement",
			path:  "/module-v1:nested-list[key='nest1']/container/containerleaf",
			value: ValueNew("!!!"),
		},
		{
			name:  "existing list entry modification",
			path:  "/module-v1:list[key='foo']/objleaf",
			value: ValueNew("!!!"),
		},
		{
			name:  "nested list entry modification",
			path:  "/module-v1:nested/list[key='foo']/objleaf",
			value: ValueNew("!!!"),
		},
		{
			name:  "nested-list list entry modification",
			path:  "/module-v1:nested-list[key='nest1']/list[key='foo']/objleaf",
			value: ValueNew("!!!"),
		},
		{
			name:  "list entry addition",
			path:  "/module-v1:list[key='idontexist']/objleaf",
			value: ValueNew("!!!"),
		},
		{
			name:  "nested list entry addition",
			path:  "/module-v1:nested/list[key='idontexist']/objleaf",
			value: ValueNew("!!!"),
		},
		{
			name:  "nested-list list entry addition",
			path:  "/module-v1:nested-list[key='nest1']/list[key='idontexist']/objleaf",
			value: ValueNew("!!!"),
		},
		{
			name:  "existing leaf-list entry modification",
			path:  "/module-v1:leaf-list[0]",
			value: ValueNew("!!!"),
		},
		{
			name:  "nested leaf-list entry modification",
			path:  "/module-v1:nested/leaf-list[1]",
			value: ValueNew("!!!"),
		},
		{
			name:  "nested-list list entry modification",
			path:  "/module-v1:nested-list[key='nest1']/leaf-list[2]",
			value: ValueNew("!!!"),
		},
		{
			name:  "leaf-list entry addition",
			path:  "/module-v1:leaf-list[7]",
			value: ValueNew("!!!"),
		},
		{
			name:  "nested list entry addition",
			path:  "/module-v1:nested/leaf-list[7]",
			value: ValueNew("!!!"),
		},
		{
			name:  "nested-list list entry addition",
			path:  "/module-v1:nested-list[key='nest1']/leaf-list[7]",
			value: ValueNew("!!!"),
		},
		{
			name:  "deeply nested entry addition",
			path:  "/module-v1:foo/bar/baz/newlist[key='idontexist']/quux/newnestedlist[0]/objleaf",
			value: ValueNew("!!!"),
		},
	}
	tree := TreeFromObject(TESTOBJ)
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			new := tree.Assoc(test.path, test.value)
			got := new.At(test.path)
			if !equal(got, test.value) {
				t.Fatalf("Assoc failed, expected %s, got %s in\n%s",
					test.value, got, new)
			}
		})
	}
}

func TestTreeDelete(t *testing.T) {
	cases := []struct {
		name string
		path string
	}{
		{
			name: "leaf",
			path: "/module-v1:container/containerleaf",
		},
		{
			name: "container",
			path: "/module-v1:container",
		},
		{
			name: "list entry leaf",
			path: "/module-v1:list[key='foo']/objleaf",
		},
		{
			name: "list entry by value",
			path: "/module-v1:list[key='foo']",
		},
		{
			name: "leaf-list entry by value",
			path: "/module-v1:leaf-list[.='2']",
		},
		{
			name: "list",
			path: "/module-v1:list",
		},
		{
			name: "leaf-list",
			path: "/module-v1:leaf-list",
		},
		//nested
		{
			name: "nested leaf",
			path: "/module-v1:nested/container/containerleaf",
		},
		{
			name: "nested container",
			path: "/module-v1:nested/container",
		},
		{
			name: "nested list entry leaf",
			path: "/module-v1:nested/list[key='foo']/objleaf",
		},
		{
			name: "nested list entry by value",
			path: "/module-v1:nested/list[key='foo']",
		},
		{
			name: "nested leaf-list entry by value",
			path: "/module-v1:nested/leaf-list[.='2']",
		},
		{
			name: "nested list",
			path: "/module-v1:nested/list",
		},
		{
			name: "nested leaf-list",
			path: "/module-v1:nested/leaf-list",
		},
		//nested list
		{
			name: "nested-list leaf",
			path: "/module-v1:nested-list[key='nested1']/container/containerleaf",
		},
		{
			name: "nested-list container",
			path: "/module-v1:nested-list[key='nested1']/container",
		},
		{
			name: "nested-list list entry leaf",
			path: "/module-v1:nested-list[key='nested1']/list[key='foo']/objleaf",
		},
		{
			name: "nested-list list entry by value",
			path: "/module-v1:nested-list[key='nested1']/list[key='foo']",
		},
		{
			name: "nested-list leaf-list entry by value",
			path: "/module-v1:nested-list[key='nested1']/leaf-list[.='2']",
		},
		{
			name: "nested-list list",
			path: "/module-v1:nested-list[key='nested1']/list",
		},
		{
			name: "nested-list leaf-list",
			path: "/module-v1:nested-list[key='nested1']/leaf-list",
		},
	}
	tree := TreeFromObject(TESTOBJ)
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			new := tree.Delete(test.path)
			if new.Contains(test.path) {
				t.Fatalf("delete failed, %s still exists in %s",
					test.path, new)
			}
		})
	}

	t.Run("list entry by position", func(t *testing.T) {
		path := "/module-v1:list[0]"
		old := tree.At(path)
		newTree := tree.Delete(path)
		new := newTree.At(path)
		if equal(old, new) {
			t.Fatalf("delete failed, %s still exists in %s",
				path, newTree)
		}
	})
	t.Run("leaf-list entry by position", func(t *testing.T) {
		path := "/module-v1:leaf-list[0]"
		old := tree.At(path)
		newTree := tree.Delete(path)
		new := newTree.At(path)
		if equal(old, new) {
			t.Fatalf("delete failed, %s still exists in %s",
				path, newTree)
		}
	})
}

func matchEditEntry(in EditEntry, entries []EditEntry) bool {
	for _, entry := range entries {
		if entry.Action == in.Action &&
			equal(entry.Path, in.Path) &&
			equal(entry.Value, in.Value) {
			return true
		}
	}
	return false
}
func TestTreeDiff(t *testing.T) {
	tree := TreeFromObject(TESTOBJ)
	cases := []struct {
		name    string
		actions []EditEntry
	}{
		{
			name: "delete",
			actions: []EditEntry{
				EditEntryNew("delete", "/module-v1:nested/container"),
			},
		}, {
			name: "assoc",
			actions: []EditEntry{
				EditEntryNew("assoc",
					"/module-v1:nested/list[0]/objleaf",
					EditEntryValue("!!!")),
			},
		}, {
			name: "assoc/delete",
			actions: []EditEntry{
				EditEntryNew("assoc",
					"/module-v1:nested/list[0]/objleaf",
					EditEntryValue("!!!")),
				EditEntryNew("delete", "/module-v1:nested/container"),
			},
		}, {
			name: "assoc new array entry",
			actions: []EditEntry{
				EditEntryNew("assoc",
					"/module-v1:leaf-list[7]",
					EditEntryValue(8)),
			},
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			new := tree
			for _, action := range test.actions {
				switch action.Action {
				case "assoc":
					new = new.assoc(action.Path,
						action.Value)
				case "delete":
					new = new.delete(action.Path)
				}
			}
			diff := tree.Diff(new)
			for _, action := range diff.Actions {
				if !matchEditEntry(action, test.actions) {
					t.Fatal("didn't find expected edit entry", action, test.actions)
				}
			}
		})
	}
	t.Run("new array is longer than old", func(t *testing.T) {
		new := tree.Assoc("/module-v1:leaf-list[7]", 8)
		diff := tree.Diff(new)
		if !equal(diff.Actions[0].Value, ValueNew(8)) {
			t.Fatal("didn't find expected diff")
		}
	})
	t.Run("new array has changed to other value", func(t *testing.T) {
		new := tree.Assoc("/module-v1:leaf-list", "!!!")
		diff := tree.Diff(new)
		if !equal(diff.Actions[0].Value, ValueNew("!!!")) {
			t.Fatal("didn't find expected diff")
		}
	})
	t.Run("new object has changed to other value", func(t *testing.T) {
		new := tree.Assoc("/module-v1:container", "!!!")
		diff := tree.Diff(new)
		if !equal(diff.Actions[0].Value, ValueNew("!!!")) {
			t.Fatal("didn't find expected diff")
		}
	})
}

func TestTreeEdit(t *testing.T) {
	tree := TreeFromObject(TESTOBJ)
	cases := []struct {
		name string
		edit *EditOperation
	}{
		{
			name: "sniff test",
			edit: EditOperationNew(
				EditEntryNew("delete",
					"/module-v1:nested/list[key='foo']"),
				EditEntryNew("delete",
					"/module-v1:nested/container"),
				EditEntryNew("assoc",
					"/module-v1:new/othercontainer/leaf",
					EditEntryValue("!!!")),
				EditEntryNew("assoc",
					"/module-v1:new/othercontainer/leaf2",
					EditEntryValue("!!!!")),
				EditEntryNew("merge",
					"/module-v1:container",
					EditEntryValue(ObjectWith(
						PairNew("containerleaf", "bar"),
						PairNew("newleaf", "baz")))),
				EditEntryNew("merge",
					"/module-v1:list",
					EditEntryValue(ArrayWith(
						ObjectWith(
							PairNew("key", "foo"),
							PairNew("objleaf", "baz"),
							PairNew("newleaf", "baz")),
						ObjectWith(
							PairNew("key", "!!!"),
							PairNew("objleaf", "!!!"),
							PairNew("newleaf", "!!!"))))),
			),
		},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			new := tree.Edit(test.edit)
			diff := tree.Diff(new)
			edited := tree.Edit(diff)
			if !equal(new, edited) {
				t.Fatalf("When editing tree:\n\t%s\nwith:\n\t%s\ngot:\n\t%s\nexpected:\n\t%s\ndifferences were:\n\t%s",
					tree.Root().AsObject(),
					diff,
					edited.Root().AsObject(),
					new.Root().AsObject(),
					new.Diff(edited))
			}
		})
	}
}

func TestTreeMarshalUnmarshal(t *testing.T) {
	tree := TreeFromObject(TESTOBJ)
	d, err := rfc7951.Marshal(tree)
	if err != nil {
		t.Fatal(err)
	}
	new := TreeNew()
	err = rfc7951.Unmarshal(d, new)
	if err != nil {
		t.Fatal(err)
	}
	if !tree.Equal(new) {
		t.Fatalf("got:\n\t%s\nexpected:\n\t%s\ndiffereneces:\n\t%s\n",
			new, tree, tree.Diff(new))
	}
}

func TestTreeMarshalEmpty(t *testing.T) {
	tree := TreeFromObject(TESTOBJ)
	d, err := rfc7951.Marshal(tree)
	if err != nil {
		t.Fatal(err)
	}
	new := new(Tree)
	err = rfc7951.Unmarshal(d, new)
	if err != nil {
		t.Fatal(err)
	}
	if !tree.Equal(new) {
		t.Fatalf("got:\n\t%s\nexpected:\n\t%s\ndiffereneces:\n\t%s\n",
			new, tree, tree.Diff(new))
	}
}

func TestTreeLength(t *testing.T) {
	tree := TreeFromObject(TESTOBJ)
	if tree.Length() != 102 {
		t.Fatal("didn't get expected length for representative tree")
	}
}

func TestTreeEqual(t *testing.T) {
	tree := TreeFromObject(TESTOBJ)
	t.Run("tree == tree", func(t *testing.T) {
		if !equal(tree, tree) {
			t.Fatal("tree didn't equal itsself")
		}
	})
	t.Run("tree != tree.Root()", func(t *testing.T) {
		if equal(tree, tree.Root()) {
			t.Fatal("tree shouldn't equal an object")
		}
	})
}

func TestTreeString(t *testing.T) {
	orig := TreeFromObject(TESTOBJ)
	str := orig.String()
	tree := TreeNew()
	err := rfc7951.Unmarshal([]byte(str), tree)
	if err != nil {
		t.Fatal(err)
	}
	if !equal(tree, orig) {
		t.Fatalf("got:\n\t%s\nexpected:\n\t%s\ndifferences:\n\t%s\n",
			tree,
			orig,
			tree.Diff(orig))
	}
}

func TestTreeFromValue(t *testing.T) {
	tree := TreeFromValue(ValueNew(TESTOBJ))
	v := tree.At("/rfc7951:data")
	got := v.ToObject()
	if got.module != "rfc7951" {
		t.Fatal("didn't get expected module name")
	}
	if !equal(got.store, TESTOBJ.store) {
		t.Fatalf("got:\n\t%s\nexpected:\n\t%s\ndifferences:\n\t%s\n",
			got,
			TESTOBJ,
			TreeFromObject(TESTOBJ).Diff(TreeFromObject(got)))
	}
}

func TestTreeFind(t *testing.T) {
	tree := TreeFromObject(TESTOBJ)
	t.Run("existing key", func(t *testing.T) {
		v, ok := tree.Find("/module-v1:container")
		if !ok || v == nil {
			t.Fatal("didn't find expected value")
		}
	})
	t.Run("non-existant key", func(t *testing.T) {
		v, ok := tree.Find("/foo:container")
		if ok || v != nil {
			t.Fatal("found unexpected value")
		}
	})
}

func TestTreeRange(t *testing.T) {
	tree := TreeFromObject(TESTOBJ)
	rangeLeaves := map[string]interface{}{
		"/module-v1:container/containerleaf":                "foo",
		"/module-v1:nested-list[0]/key":                     "nest1",
		"/module-v1:nested-list[0]/container/containerleaf": "foo",
		"/module-v1:nested-list[0]/leaf-list[0]":            1,
		"/module-v1:nested-list[0]/leaf-list[1]":            2,
		"/module-v1:nested-list[0]/leaf-list[2]":            3,
		"/module-v1:nested-list[0]/leaf-list[3]":            4,
		"/module-v1:nested-list[0]/leaf-list[4]":            5,
		"/module-v1:nested-list[0]/leaf-list[5]":            6,
		"/module-v1:nested-list[0]/leaf-list[6]":            7,
		"/module-v1:nested-list[0]/list[0]/objleaf":         "bar",
		"/module-v1:nested-list[0]/list[0]/key":             "foo",
		"/module-v1:nested-list[0]/list[1]/objleaf":         "baz",
		"/module-v1:nested-list[0]/list[1]/key":             "bar",
		"/module-v1:nested-list[0]/list[2]/key":             "baz",
		"/module-v1:nested-list[0]/list[2]/objleaf":         "quux",
		"/module-v1:nested-list[0]/list[3]/key":             "quux",
		"/module-v1:nested-list[0]/list[3]/objleaf":         "quuz",
		"/module-v1:nested-list[0]/leaf":                    "foo",
		"/module-v1:nested-list[1]/list[0]/objleaf":         "bar",
		"/module-v1:nested-list[1]/list[0]/key":             "foo",
		"/module-v1:nested-list[1]/list[1]/key":             "bar",
		"/module-v1:nested-list[1]/list[1]/objleaf":         "baz",
		"/module-v1:nested-list[1]/list[2]/key":             "baz",
		"/module-v1:nested-list[1]/list[2]/objleaf":         "quux",
		"/module-v1:nested-list[1]/list[3]/key":             "quux",
		"/module-v1:nested-list[1]/list[3]/objleaf":         "quuz",
		"/module-v1:nested-list[1]/container/containerleaf": "foo",
		"/module-v1:nested-list[1]/leaf-list[0]":            1,
		"/module-v1:nested-list[1]/leaf-list[1]":            2,
		"/module-v1:nested-list[1]/leaf-list[2]":            3,
		"/module-v1:nested-list[1]/leaf-list[3]":            4,
		"/module-v1:nested-list[1]/leaf-list[4]":            5,
		"/module-v1:nested-list[1]/leaf-list[5]":            6,
		"/module-v1:nested-list[1]/leaf-list[6]":            7,
		"/module-v1:nested-list[1]/key":                     "nest2",
		"/module-v1:nested-list[1]/leaf":                    "foo",
		"/module-v1:nested/container/containerleaf":         "foo",
		"/module-v1:nested/leaf":                            "foo",
		"/module-v1:nested/list[0]/objleaf":                 "bar",
		"/module-v1:nested/list[0]/key":                     "foo",
		"/module-v1:nested/list[1]/objleaf":                 "baz",
		"/module-v1:nested/list[1]/key":                     "bar",
		"/module-v1:nested/list[2]/objleaf":                 "quux",
		"/module-v1:nested/list[2]/key":                     "baz",
		"/module-v1:nested/list[3]/key":                     "quux",
		"/module-v1:nested/list[3]/objleaf":                 "quuz",
		"/module-v1:nested/leaf-list[0]":                    1,
		"/module-v1:nested/leaf-list[1]":                    2,
		"/module-v1:nested/leaf-list[2]":                    3,
		"/module-v1:nested/leaf-list[3]":                    4,
		"/module-v1:nested/leaf-list[4]":                    5,
		"/module-v1:nested/leaf-list[5]":                    6,
		"/module-v1:nested/leaf-list[6]":                    7,
		"/module-v1:list[0]/key":                            "foo",
		"/module-v1:list[0]/objleaf":                        "bar",
		"/module-v1:list[1]/objleaf":                        "baz",
		"/module-v1:list[1]/key":                            "bar",
		"/module-v1:list[2]/objleaf":                        "quux",
		"/module-v1:list[2]/key":                            "baz",
		"/module-v1:list[3]/key":                            "quux",
		"/module-v1:list[3]/objleaf":                        "quuz",
		"/module-v1:leaf-list[0]":                           1,
		"/module-v1:leaf-list[1]":                           2,
		"/module-v1:leaf-list[2]":                           3,
		"/module-v1:leaf-list[3]":                           4,
		"/module-v1:leaf-list[4]":                           5,
		"/module-v1:leaf-list[5]":                           6,
		"/module-v1:leaf-list[6]":                           7,
		"/module-v1:leaf":                                   "foo",
	}
	t.Run("func(*InstanceID, *Value)", func(t *testing.T) {
		count := 0
		tree.Range(func(iid *InstanceID, v *Value) {
			v.Perform(func(o *Object) {
			}, func(a *Array) {
			}, func(other interface{}) {
				count++
				if !equal(ValueNew(rangeLeaves[iid.String()]), v) {
					t.Fatal("didn't get expected value for",
						iid, rangeLeaves[iid.String()], v)
				}
			})
		})
		if count != len(rangeLeaves) {
			t.Fatal("didn't access all the values")
		}
	})
	t.Run("func(*InstanceID, *Value) bool", func(t *testing.T) {
		count := 0
		tree.Range(func(iid *InstanceID, v *Value) bool {
			return v.Perform(func(o *Object) bool {
				return true
			}, func(a *Array) bool {
				return true
			}, func(other interface{}) bool {
				if iid.String() == "/module-v1:leaf-list[2]" {
					return false
				}
				count++
				if !equal(ValueNew(rangeLeaves[iid.String()]), v) {
					t.Fatal("didn't get expected value for",
						iid, rangeLeaves[iid.String()], v)
				}
				return true
			}).(bool)
		})
		if count == len(rangeLeaves) {
			t.Fatal("accessed too many values")
		}
	})
	t.Run("func(string, *Value)", func(t *testing.T) {
		count := 0
		tree.Range(func(iid string, v *Value) {
			v.Perform(func(o *Object) {
			}, func(a *Array) {
			}, func(other interface{}) {
				count++
				if !equal(ValueNew(rangeLeaves[iid]), v) {
					t.Fatal("didn't get expected value for",
						iid, rangeLeaves[iid], v)
				}
			})
		})
		if count != len(rangeLeaves) {
			t.Fatal("didn't access all the values")
		}
	})
	t.Run("func(string, *Value) bool", func(t *testing.T) {
		count := 0
		tree.Range(func(iid string, v *Value) bool {
			return v.Perform(func(o *Object) bool {
				return true
			}, func(a *Array) bool {
				return true
			}, func(other interface{}) bool {
				if iid == "/module-v1:leaf-list[2]" {
					return false
				}
				count++
				if !equal(ValueNew(rangeLeaves[iid]), v) {
					t.Fatal("didn't get expected value for",
						iid, rangeLeaves[iid], v)
				}
				return true
			}).(bool)
		})
		if count == len(rangeLeaves) {
			t.Fatal("accessed too many values")
		}
	})
	t.Run("func(*InstanceID)", func(t *testing.T) {
		count := 0
		tree.Range(func(iid *InstanceID) {
			count++
		})
		if count < len(rangeLeaves) {
			t.Fatal("didn't access all the values")
		}
	})
	t.Run("func(*InstanceID) bool", func(t *testing.T) {
		count := 0
		tree.Range(func(iid *InstanceID) bool {
			if iid.String() == "/module-v1:leaf-list[2]" {
				return false
			}
			count++
			return true
		})
		if count == len(rangeLeaves) {
			t.Fatal("accessed too many values")
		}
	})
	t.Run("func(string)", func(t *testing.T) {
		count := 0
		tree.Range(func(iid string) {
			count++
		})
		if count < len(rangeLeaves) {
			t.Fatal("didn't access all the values")
		}
	})
	t.Run("func(string) bool", func(t *testing.T) {
		count := 0
		tree.Range(func(iid string) bool {
			if iid == "/module-v1:leaf-list[2]" {
				return false
			}
			count++
			return true
		})
		if count == len(rangeLeaves) {
			t.Fatal("accessed too many values")
		}
	})
	t.Run("func(*Value)", func(t *testing.T) {
		count := 0
		tree.Range(func(v *Value) {
			v.Perform(func(o *Object) {
			}, func(a *Array) {
			}, func(other interface{}) {
				count++
			})
		})
		if count != len(rangeLeaves) {
			t.Fatal("didn't access all the values")
		}
	})
	t.Run("func(*Value) bool", func(t *testing.T) {
		count := 0
		tree.Range(func(v *Value) bool {
			return v.Perform(func(o *Object) bool {
				return true
			}, func(a *Array) bool {
				return false
			}, func(other interface{}) bool {
				count++
				return true
			}).(bool)
		})
		if count == len(rangeLeaves) {
			t.Fatal("accessed too many values")
		}
	})
	t.Run("func(*Value) bool object", func(t *testing.T) {
		count := 0
		tree.Range(func(v *Value) bool {
			return v.Perform(func(o *Object) bool {
				return false
			}, func(a *Array) bool {
				return true
			}, func(other interface{}) bool {
				count++
				return true
			}).(bool)
		})
		if count == len(rangeLeaves) {
			t.Fatal("accessed too many values")
		}
	})
}
