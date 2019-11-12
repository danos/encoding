// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

func assert(expr bool, ifFalse func()) {
	if !expr {
		ifFalse()
	}
}
