// Copyright (c) 2017-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

// RFC7951 specific tests to verify correct behaviour of boolean types
// with and without the emptyleaf tag.
//
// A boolean type that is has an emptyleaf tag should be encoded as
// [null], as per RFC 7951, Sec 6.9
//
package rfc7951

import (
	"reflect"
	"strings"
	"testing"
)

type EmptyLeafs struct {
	Ba bool `rfc7951:"ba,emptyleaf,string"`
	Bb bool `rfc7951:"bb,emptyleaf"`
	Bc bool `rfc7951:"bc,emptyleaf"`
	Bd bool `rfc7951:"bd"`
	Be bool `rfc7951:"be"`
	Bf bool `rfc7951:"bf"`
	Bg bool `rfc7951:"bg"`
	Bh bool `rfc7951:"bh,emptyleaf,omitempty"`
	Bi bool `rfc7951:"bi,emptyleaf,omitempty"`
	Bj bool `rfc7951:"bj,emptyleaf,omitempty"`
	Bk bool `rfc7951:"bk,omitempty"`
	Bl bool `rfc7951:"bl,omitempty"`
	Bm bool `rfc7951:"bm,omitempty"`
	Bn bool `rfc7951:"bn,omitempty"`

	Pa *bool `rfc7951:"pa,emptyleaf"`
	Pb *bool `rfc7951:"pb,emptyleaf"`
	Pc *bool `rfc7951:"pc,emptyleaf"`
	Pd *bool `rfc7951:"pd"`
	Pe *bool `rfc7951:"pe"`
	Pf *bool `rfc7951:"pf"`
	Pg *bool `rfc7951:"pg"`
	Ph *bool `rfc7951:"ph,emptyleaf,omitempty"`
	Pi *bool `rfc7951:"pi,emptyleaf,omitempty"`
	Pj *bool `rfc7951:"pj,emptyleaf,omitempty"`
	Pk *bool `rfc7951:"pk,omitempty"`
	Pl *bool `rfc7951:"pl,omitempty"`
	Pm *bool `rfc7951:"pm,omitempty"`
	Pn *bool `rfc7951:"pn,omitempty"`
}

var expectedEmptyLeafEncode = `{
 "ba": [
  null
 ],
 "bd": true,
 "be": false,
 "bf": false,
 "bg": false,
 "bi": [
  null
 ],
 "bk": true,
 "pa": [
  null
 ],
 "pd": true,
 "pe": false,
 "pf": null,
 "pg": null,
 "pi": [
  null
 ],
 "pk": true,
 "pl": false
}`

var emptyLeafDecode = `{
 "ba": [null],
 "bb": null,
 "bd": true,
 "be": false,
 "bf": null,
 "bh": null,
 "bi": [null],
 "bk": true,
 "bl": false,
 "bm": null,
 "pa": [null],
 "pb": null,
 "pd": true,
 "pe": false,
 "pf": null,
 "ph": null,
 "pi": [null],
 "pk": true,
 "pl": false,
 "pm": null
}`

func TestEmptyLeafEncode(t *testing.T) {
	var e EmptyLeafs
	err := Unmarshal([]byte(emptyLeafDecode), &e)
	if err != nil {
		t.Fatal(err)
	}

	got, err := MarshalIndent(&e, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != expectedEmptyLeafEncode {
		t.Fatalf(" got: %s\nwant: %s\n", string(got), expectedEmptyLeafEncode)
	}
}

func TestEmptyLeafDecode(t *testing.T) {
	var e EmptyLeafs
	var d EmptyLeafs

	err := Unmarshal([]byte(emptyLeafDecode), &e)
	if err != nil {
		t.Fatal(err)
	}

	var negative bool
	positive := true
	d.Ba = true
	d.Bd = true
	d.Bi = true
	d.Bk = true
	d.Pa = &positive
	d.Pd = &positive
	d.Pe = &negative
	d.Pi = &positive
	d.Pk = &positive
	d.Pl = &negative
	if reflect.DeepEqual(e, d) == false {
		t.Errorf("EmptyLeaf decode not as expected\n Got: %+v\n Want: %+v", e, d)
	}
}

func checkError(t *testing.T, err error, content string) {

	if err == nil {
		t.Fatalf("Expected an error containing: %s", content)
	}
	if !strings.Contains(err.Error(), content) {
		t.Fatal(err)
	}
}

var emptyLeafTooBig = `{
 "ba": [
  null, null
 ]
}`

func TestEmptyLeafTooBig(t *testing.T) {
	var el EmptyLeafs

	err := Unmarshal([]byte(emptyLeafTooBig), &el)
	checkError(t, err, "malformed empty leaf")
}

var notAnEmptyLeaf = `{
 "ba": [true]
}`

func TestNotAnEmptyLeaf(t *testing.T) {
	var el EmptyLeafs

	err := Unmarshal([]byte(notAnEmptyLeaf), &el)
	checkError(t, err, "invalid empty leaf")
}

var boolNotTakeEmptyLeaf = `{
 "be": [null]
}`

func TestBoolNotTakeEmptyLeaf(t *testing.T) {
	var el EmptyLeafs

	err := Unmarshal([]byte(notAnEmptyLeaf), &el)
	checkError(t, err, "invalid empty leaf")
}

var quotedIsNotAnEmptyLeaf = `{
 "ba": "[null]"
}`

func TestQuotedIsNotAnEmptyLeaf(t *testing.T) {
	var el EmptyLeafs

	err := Unmarshal([]byte(quotedIsNotAnEmptyLeaf), &el)
	checkError(t, err, "invalid empty leaf")
}

var aBoolIsNotAnEmptyLeaf = `{
 "ba": true
}`

func TestBoolIsNotAnEmptyLeaf(t *testing.T) {
	var el EmptyLeafs

	err := Unmarshal([]byte(aBoolIsNotAnEmptyLeaf), &el)
	checkError(t, err, "invalid empty leaf")
}

var emptyArrayIsNotAnEmptyLeaf = `{
 "ba": []
}`

func TestEmptyArrayIsNotAnEmptyLeaf(t *testing.T) {
	var el EmptyLeafs

	err := Unmarshal([]byte(emptyArrayIsNotAnEmptyLeaf), &el)
	checkError(t, err, "malformed empty leaf")
}

type DecoderBooleans struct {
	Br uint8 `rfc7951:"br,emptyleaf"`
}

var uint8IsNotAnEmptyLeaf = `{
 "br": 123
}`

func TestEmptyLeafIgnoredNonBool(t *testing.T) {
	var db DecoderBooleans

	err := Unmarshal([]byte(uint8IsNotAnEmptyLeaf), &db)
	if err != nil {
		t.Fatal(err)
	}
}
