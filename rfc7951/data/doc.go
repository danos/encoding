// Copyright (c) 2019, AT&T Intellectual Property.
//
// SPDX-License-Identifier: MPL-2.0

// Package data implements an convenient object model for interacting with
// arbitrary rfc7951 data. The Trees, Objects, and Arrays in this library
// are immutable. This means that updating the structure will yield a
// new copy with the changes made, this is made efficient by sharing
// much of the structure of the new object with the old one. The library
// is based on the central Value type that holds arbitrary RFC7951 data
// this may take on Object, Array, int types, uint types, strings,
// float types, bools, the empty value, and nil. This may be thought of
// as a restricted form of the go interface{} type. The provided Tree
// type is a special form of Object that allows for complex operations
// based on instance-identifier paths.
package data
