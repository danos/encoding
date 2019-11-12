// Copyright (c) 2019, AT&T Intellectual Property. All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0
package data

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/danos/encoding/rfc7951"
)

func ExampleEditOperation_marshal() {
	edit := EditOperation{
		Actions: []EditEntry{
			{
				Action: EditAssoc,
				Path:   InstanceIDNew("/module-v1:foo/bar"),
				Value: ValueNew(ObjectWith(
					PairNew("bar", "quuz"))),
			},
		},
	}
	enc := rfc7951.NewEncoder(os.Stdout)
	err := enc.Encode(&edit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	// Output: {"actions":[{"action":"assoc","path":"/module-v1:foo/bar","value":{"bar":"quuz"}}]}
}

func ExampleEditOperation_string() {
	edit := EditOperation{
		Actions: []EditEntry{
			{
				Action: EditAssoc,
				Path:   InstanceIDNew("/module-v1:foo/bar"),
				Value: ValueNew(ObjectWith(
					PairNew("bar", "quuz"))),
			},
		},
	}
	fmt.Println(edit.String())
	// Output: {"actions":[{"action":"assoc","path":"/module-v1:foo/bar","value":{"bar":"quuz"}}]}
}

func TestEditOperationMarshal(t *testing.T) {
	t.Run("handles bogus action", func(t *testing.T) {
		edit := EditOperation{
			Actions: []EditEntry{
				{
					Action: EditAssoc,
					Path:   InstanceIDNew("/module-v1:foo/bar"),
					Value: ValueNew(ObjectWith(
						PairNew("bar", "quuz"))),
				},
				{
					Action: "Bogus!",
					Path:   InstanceIDNew("/module-v1:foo/bar"),
					Value: ValueNew(ObjectWith(
						PairNew("bar", "quuz"))),
				},
			},
		}
		enc := rfc7951.NewEncoder(os.Stdout)
		err := enc.Encode(&edit)
		if err == nil {
			t.Fatal("didn't get expected error")
		}
	})
}
func ExampleEditOperation_unmarshal() {
	var edit EditOperation
	s := `{
		"actions":[
			{
				"action":"assoc",
				"path":"/module-v1:foo/bar",
				"value":{"bar":"quuz"}
			},
			{
				"action":"delete",
				"path":"/module-v1:foo/bar"
			},
			{
				"action":"merge",
				"path":"/module-v1:foo/bar",
				"value":{"bar":"quux"}
			}
		]
	}`
	dec := rfc7951.NewDecoder(strings.NewReader(s))
	err := dec.Decode(&edit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	enc := rfc7951.NewEncoder(os.Stdout)
	err = enc.Encode(&edit)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	// Output: {"actions":[{"action":"assoc","path":"/module-v1:foo/bar","value":{"bar":"quuz"}},{"action":"delete","path":"/module-v1:foo/bar"},{"action":"merge","path":"/module-v1:foo/bar","value":{"bar":"quux"}}]}
}

func TestEditOperationUnmarshal(t *testing.T) {
	t.Run("handles bogus action", func(t *testing.T) {
		var edit EditOperation
		s := `{
			"actions":[
				{
					"action":"assoc",
					"path":"/module-v1:foo/bar",
					"value":{"bar":"quuz"}
				},
				{
					"action":"bogus!",
					"path":"/module-v1:foo/bar",
					"value":{"bar":"quuz"}
				}
			]
		}`
		dec := rfc7951.NewDecoder(strings.NewReader(s))
		err := dec.Decode(&edit)
		if err == nil {
			t.Fatal("didn't get expected error")
		}
	})
	t.Run("handles non string action", func(t *testing.T) {
		var edit EditOperation
		s := `{
			"actions":[
				{
					"action":"assoc",
					"path":"/module-v1:foo/bar",
					"value":{"bar":"quuz"}
				},
				{
					"action":10,
					"path":"/module-v1:foo/bar",
					"value":{"bar":"quuz"}
				}
			]
		}`
		dec := rfc7951.NewDecoder(strings.NewReader(s))
		err := dec.Decode(&edit)
		if err == nil {
			t.Fatal("didn't get expected error")
		}
	})
}
