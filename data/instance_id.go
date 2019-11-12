// Copyright (c) 2018-2019, AT&T Intellectual Property.
// All rights reserved.
//
// SPDX-License-Identifier: MPL-2.0

package data

import (
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/danos/encoding/rfc7951"
)

const (
	sp   = " "
	htab = "	"
	wsp  = sp + htab
)

// InstanceIDNew parses an instance identifier string into an InstanceID object
func InstanceIDNew(instance string) *InstanceID {
	return (&InstanceID{}).parse(instance)
}

// InstanceID is an RFC7951 instance-identifier type.
// It is defined here https://tools.ietf.org/html/rfc7951#section-6.11
//
// RFC7951 instance identifiers match the following grammar:
//     instance-identifier = 1*("/" (node-identifier *predicate))
//     predicate           = "[" *WSP (predicate-expr / pos) *WSP "]"
//     predicate-expr      = (node-identifier / ".") *WSP "=" *WSP
//                           ((DQUOTE string DQUOTE) /
//                            (SQUOTE string SQUOTE))
//     pos                 = non-negative-integer-value
//     node-identifier     = [prefix ":"] identifier
//     identifier          = (ALPHA / "_")
//                           *(ALPHA / DIGIT / "_" / "-" / ".")
//     prefix              = identifier
//     non-negative-integer-value = "0" / positive-integer-value
//     positive-integer-value = (non-zero-digit *DIGIT)
//     string              = < an unquoted string as returned by the scanner >
//     non-zero-digit      = %x31-39
//     DIGIT               = %x30-39
//                           ; 0-9
//     ALPHA               = %x41-5A / %x61-7A
//                           ; A-Z / a-z
//     WSP                 = SP / HTAB
//                           ; whitespace
//     DQUOTE              = %x22
//                           ; " (Double Quote)
//     SQUOTE              = %x27
//                           ; ' (Single Quote)
type InstanceID struct {
	ids []*nodeID
}

// path returns the path of the instance ID up to the last fully
// addressable node. Selector can be called on this to get a filter
// to match the final element against.
func (i *InstanceID) path() *InstanceID {
	if len(i.ids) == 0 {
		return nil
	}
	last := i.ids[len(i.ids)-1]
	if last.predicates == nil {
		return &InstanceID{
			ids: i.ids[:len(i.ids)-1],
		}
	}
	out := i.copy()
	last = out.ids[len(out.ids)-1]
	last.predicates = nil
	return out
}

// copy returns a copy of the instance-identifier that can subsequently
// be modified without effecting the original.
func (i *InstanceID) copy() *InstanceID {
	newIds := make([]*nodeID, len(i.ids))
	for i, v := range i.ids {
		newIds[i] = v.copy()
	}
	return &InstanceID{
		ids: newIds,
	}
}

func (id *nodeID) copy() *nodeID {
	new := *id
	new.predicates = id.predicates.copy()
	return &new
}

func (p *predicates) copy() *predicates {
	if p == nil {
		return nil
	}
	out := make([]*predicate, len(p.preds))
	for i, pred := range p.preds {
		out[i] = pred.copy()
	}
	return &predicates{preds: out}
}

func (p *predicate) copy() *predicate {
	new := *p
	return &new
}

// UnmarshalRFC7951 will parse an instance-identifier that
// was received via an RFC7951 encoded message.
func (i *InstanceID) UnmarshalRFC7951(msg []byte) (err error) {
	defer func() {
		r := recover()
		switch v := r.(type) {
		case error:
			err = v
			return
		}
	}()
	var s string
	err = rfc7951.Unmarshal(msg, &s)
	if err != nil {
		return err
	}
	i.parse(s)
	return nil
}

// MarshalRFC7951 will serialize the instance-identifier into
// an RFC7951 compatible format.
func (i *InstanceID) MarshalRFC7951() ([]byte, error) {
	return []byte("\"" + i.String() + "\""), nil
}

// Equal determines if two instance-identifiers are the same.
// It implements a common equality interface so other must be
// interface{}.
func (i *InstanceID) Equal(other interface{}) bool {
	oi, isInstanceID := other.(*InstanceID)
	return isInstanceID &&
		oi.String() == i.String()
}

func (i *InstanceID) push(nodeIDstring string) *InstanceID {
	out := i.copy()
	var prefix string
	if len(i.ids) != 0 {
		prev := out.ids[len(out.ids)-1]
		prefix = prev.prefix
	}
	node := (&nodeID{}).parse(prefix, nodeIDstring)
	out.ids = append(out.ids, node)
	return out
}

func (i *InstanceID) addPosPredicate(pos int) *InstanceID {
	out := i.copy()
	if len(out.ids) == 0 {
		return i
	}
	last := out.ids[len(out.ids)-1]
	if last.predicates == nil {
		last.predicates = &predicates{}
	}
	last.predicates.preds = append(last.predicates.preds, &predicate{
		instanceIDSelector: &posPredicate{uint64(pos)},
	})
	return out
}

type instanceIDSelector interface {
	Find(*Value) (*Value, bool)
	computeIdentifier(*Value) interface{}
	computeIdentifierDefault(*Value) interface{}
}

type nodeID struct {
	prefix, identifier string
	prefixInferred     bool
	predicates         *predicates
}

type predicates struct {
	preds []*predicate
}

type predicate struct {
	instanceIDSelector
}

type posPredicate struct {
	pos uint64
}

type exprPredicate struct {
	nodeID *nodeID
	value  string
}

// stringer exists so we don't need to import fmt for the definition
// in general this is what interfaces are good for.
type stringer interface {
	String() string
}

// parse implements a straight forward recursive descent parser for the
// RFC7951 instance identifier grammar. Using lex/yacc for this would be
// overkill so just parse the nodes inline to build a matcher.
func (i *InstanceID) parse(input string) *InstanceID {
	// instance-identifier = 1*("/" (node-identifier *predicate))
	defer func() {
		errstr := "invalid instance identifier"
		v := recover()
		if v == nil {
			return
		}
		switch v := v.(type) {
		case string:
			errstr += ": " + v
		case error:
			errstr += ": " + v.Error()
		case stringer:
			errstr += ": " + v.String()
		}
		panic(errors.New(errstr))
	}()

	nodeIDstrings := i.getNodeIDStrings(input)
	if len(nodeIDstrings) == 0 {
		panic("must specify at least one node-identifier")
	}
	if nodeIDstrings[0] != "" {
		panic("must start with a \"/\"")
	}
	nodeIDstrings = nodeIDstrings[1:]
	if len(nodeIDstrings) == 0 {
		panic("must specify at least one node-identifier")
	}
	nodeIDs := make([]*nodeID, 0, len(nodeIDstrings))
	node := &nodeID{}
	for _, nodeIDstring := range nodeIDstrings {
		prefix := node.prefix
		node = &nodeID{}
		node.parse(prefix, nodeIDstring)
		nodeIDs = append(nodeIDs, node)
	}
	i.ids = nodeIDs

	return i
}

func (i *InstanceID) getNodeIDStrings(input string) []string {
	var inSingleQ, inDoubleQ bool
	var out []string
	var first int
	for i, r := range input {
		switch r {
		case '\'':
			inSingleQ = !inSingleQ
		case '"':
			inDoubleQ = !inDoubleQ
		case '/':
			if !inDoubleQ && !inSingleQ {
				out = append(out, input[first:i])
				first = i + 1
			}
		default:
		}
	}
	if first < len(input) {
		out = append(out, input[first:len(input)])
	}
	if inDoubleQ || inSingleQ {
		panic("unterminated quote")
	}
	return out
}

func (id *nodeID) parse(prefix, input string) *nodeID {
	// (node-identifier *predicate)
	// node-identifier     = [prefix ":"] identifier
	// predicate           = "[" *WSP (predicate-expr / pos) *WSP "]"
	idParts := strings.SplitN(input, ":", 2)
	switch len(idParts) {
	case 1:
		id.identifier = idParts[0]
		if prefix != "" {
			id.prefix = prefix
			id.prefixInferred = true
		} else {
			panic("unable to determine prefix")
		}
	case 2:
		id.prefix, id.identifier = idParts[0], idParts[1]
		if id.prefix == prefix {
			id.prefixInferred = true
		}
	}
	id.checkIDPart(id.prefix)
	if strings.ContainsRune(id.identifier, '[') {
		predsStart := strings.IndexRune(id.identifier, '[')
		predString := id.identifier[predsStart:]
		id.identifier = id.identifier[:predsStart]
		id.predicates = (&predicates{}).parse(id.prefix, predString)
	}
	id.checkIDPart(id.identifier)
	return id
}

func (id *nodeID) checkIDPart(str string) {
	// identifier          = (ALPHA / "_")
	//                 *(ALPHA / DIGIT / "_" / "-" / ".")
	errInval := errors.New("invalid node-identifier " + str)

	if len(str) >= 3 {
		if strings.ToUpper(str[:3]) == "XML" {
			panic(errors.New("invalid identifier," +
				" not allowed to start with xml: " + str))
		}
	}
	for i, r := range str {
		if i == 0 {
			if !(r == '_' || unicode.IsLetter(r)) {
				panic(errInval)
			}
		} else if !id.isAlphaNumeric(r) && r != '-' && r != '.' {
			panic(errInval)
		}
	}
}

func (id *nodeID) isAlphaNumeric(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

func (p *predicates) parse(prefix, input string) *predicates {
	predStrings := p.getPredicateStrings(input)
	for _, predString := range predStrings {
		p.preds = append(p.preds,
			(&predicate{}).parse(prefix, predString))
	}
	return p
}

func (p *predicates) getPredicateStrings(input string) []string {
	var inSingleQ, inDoubleQ, inPredicate bool
	var out []string
	var first int
	for i, r := range input {
		switch r {
		case '[':
			if !inDoubleQ && !inSingleQ {
				if inPredicate {
					panic("nested predicates are not allowed")
				}
				inPredicate = true
			}
		case ']':
			if !inDoubleQ && !inSingleQ {
				out = append(out, input[first:i+1])
				first = i + 1
				inPredicate = false
			}
		case '\'':
			inSingleQ = !inSingleQ
		case '"':
			inDoubleQ = !inDoubleQ
		default:
		}
	}
	if inDoubleQ || inSingleQ {
		panic("unterminated quote")
	}
	if inPredicate {
		panic("unterminated predicate")
	}
	return out
}

func (p *predicate) parse(prefix, input string) *predicate {
	// predicate           = "[" *WSP (predicate-expr / pos) *WSP "]"
	if input[0] != '[' || input[len(input)-1] != ']' {
		panic("invalid predicate \"" + input + "\"")
	}
	input = strings.Trim(input, "[]")
	input = strings.Trim(input, wsp)
	_, err := strconv.ParseUint(input, 10, 64)
	if err == nil {
		p.instanceIDSelector = (&posPredicate{}).parse(prefix, input)
	} else {
		p.instanceIDSelector = (&exprPredicate{}).parse(prefix, input)
	}

	return p
}

func (p *posPredicate) parse(prefix, input string) *posPredicate {
	// pos                 = non-negative-integer-value
	u, err := strconv.ParseUint(input, 10, 64)
	if err != nil {
		panic(err)
	}
	p.pos = u
	return p
}

func (p *exprPredicate) parse(prefix, input string) *exprPredicate {
	// predicate-expr      = (node-identifier / ".") *WSP "=" *WSP
	//                         ((DQUOTE string DQUOTE) /
	//                          (SQUOTE string SQUOTE))
	exprParts := strings.SplitN(input, "=", 2)
	if len(exprParts) < 2 {
		panic("invalid predicate expression " + input)
	}
	for i, v := range exprParts {
		exprParts[i] = strings.Trim(v, wsp)
	}
	if exprParts[0] == "." {
		p.nodeID = &nodeID{
			prefix:         prefix,
			prefixInferred: true,
			identifier:     ".",
		}
	} else {
		p.nodeID = (&nodeID{}).parse(prefix, exprParts[0])
	}
	expr := exprParts[1]
	var end int
	switch expr[0] {
	case '"':
		end = strings.IndexRune(expr[1:], '"')
	case '\'':
		end = strings.IndexRune(expr[1:], '\'')
	default:
		panic("invalid predicate, expected ''' or '\"'")
	}
	expr = expr[1:]
	if end != len(expr)-1 {
		panic("unterminated expression value")
	}
	expr = expr[0:end]
	p.value = expr
	return p
}

// String will format an instance-identifier as a string.
// This instance-identifier is normalized to the RFC7951 spec.
func (i *InstanceID) String() string {
	ss := make([]string, 0, len(i.ids))
	for _, id := range i.ids {
		ss = append(ss, id.String())
	}
	return "/" + strings.Join(ss, "/")
}

func (id *nodeID) String() string {
	if id.prefix != "" && !id.prefixInferred {
		return id.prefix + ":" + id.identifier + id.predicates.String()
	}
	return id.identifier + id.predicates.String()
}

func (p *predicates) String() string {
	if p == nil {
		return ""
	}
	var out string
	for _, pred := range p.preds {
		out = out + pred.String()
	}
	return out
}

func (p *predicate) String() string {
	if p == nil {
		return ""
	}
	return "[" + p.instanceIDSelector.(stringer).String() + "]"
}

func (p *posPredicate) String() string {
	return strconv.FormatUint(p.pos, 10)
}

func (p *exprPredicate) String() string {
	return p.nodeID.String() + "=" + "'" + p.value + "'"
}

// RFC7951String implements string conversion as expected by the value type.
// RFC7951String is different than stringer in case the types need to encode
// differently for the encoding format.
func (i *InstanceID) RFC7951String() string {
	return i.String()
}

// Find will traverse the tree to find the Value
// to which the instance-identifier refers.
func (i *InstanceID) Find(value *Value) (*Value, bool) {
	var found bool
	for _, nodeID := range i.ids {
		value, found = nodeID.Find(value)
		if !found {
			return nil, false
		}
	}
	return value, found
}

func (id *nodeID) Find(value *Value) (*Value, bool) {
	if value == nil {
		return nil, false
	}
	var found bool
	out := ValueNew(value.Perform(func(coll *Object) *Value {
		value = coll.At(id.prefix + ":" + id.identifier)
		found = coll.Contains(id.prefix + ":" + id.identifier)
		if id.predicates != nil {
			value, found = id.predicates.Find(value)
		}
		return value
	}))
	return out, found
}

func (p *predicates) Find(value *Value) (*Value, bool) {
	var out *Value
	cur := value
	found := true
	for i, pred := range p.preds {
		cur, found = pred.Find(cur)
		if !found {
			return nil, found
		}
		out = ValueNew(cur.Perform(
			func(arr *Array) *Value {
				return ValueNew(arr)
			},
			func(v interface{}) *Value {
				if i != len(p.preds)-1 {
					found = false
					return nil
				}
				return ValueNew(v)
			}))
		if out == nil {
			break
		}
	}
	ret := ValueNew(out.Perform(
		func(arr *Array) *Value {
			if arr.Length() != 1 {
				found = false
				return nil //no exact match
			}
			return arr.At(0)
		},
		func(v *Value) *Value {
			return v
		}))
	return ret, found
}

func (p *posPredicate) Find(value *Value) (*Value, bool) {
	var found bool
	out := ValueNew(value.Perform(func(arr *Array) *Value {
		found = arr.Contains(int(p.pos))
		if !found {
			return nil
		}
		found = true
		return arr.At(int(p.pos))
	}))
	return out, found
}

func (p *exprPredicate) Find(value *Value) (*Value, bool) {
	var found bool
	out := ValueNew(value.Perform(func(a *Array) *Value {
		if p.nodeID.identifier == "." {
			//only leaf-lists can be referenced this way
			return a.detect(func(value *Value) bool {
				found = value.RFC7951String() == p.value
				return found
			})
		}
		//only lists can be referenced this way
		return ValueNew(a.selectItems(func(value *Value) bool {
			value, foundSelector := p.nodeID.Find(value)
			if !foundSelector {
				return false
			}
			matched := value.RFC7951String() == p.value
			found = found || matched
			return matched
		}))
	}))
	return out, found
}

// MatchAgainst returns the value at the location represented
// by the instance-identifier. If none, it returns nil.
func (i *InstanceID) MatchAgainst(value *Value) *Value {
	v, _ := i.Find(value)
	return v
}

func (i *InstanceID) selector() instanceIDSelector {
	if len(i.ids) == 0 {
		return nil
	}
	return i.ids[len(i.ids)-1].selector()
}

func (id *nodeID) selector() instanceIDSelector {
	if id.predicates == nil {
		return id
	}
	return id.predicates
}

func (id *nodeID) computeIdentifier(value *Value) interface{} {
	return value.Perform(func(o *Object) interface{} {
		key := id.prefix + ":" + id.identifier
		if o.Contains(key) {
			return key
		}
		return nil
	})
}

func (p *predicates) computeIdentifier(value *Value) interface{} {
	return value.Perform(func(a *Array) interface{} {
		// Start with all indicies matched
		matched := make(map[int]struct{})
		for i := 0; i < a.Length(); i++ {
			matched[i] = struct{}{}
		}
		for _, pred := range p.preds {
			id := pred.computeIdentifier(value)
			if id == nil {
				matched = map[int]struct{}{}
				break
			}
			switch v := id.(type) {
			case []int:
				// We got more than one match, filter the
				// previously matched indicies based on the
				// ones matched by the current predicate.
				got := make(map[int]struct{})
				for _, id := range v {
					_, seen := matched[id]
					if seen {
						got[id] = struct{}{}
					}
				}
				matched = got
			case int:
				got := make(map[int]struct{})
				_, seen := matched[v]
				if seen {
					got[v] = struct{}{}
				}
				matched = got
			}

		}
		// If we fully matched more than one index then the id
		// is not valid
		if len(matched) > 1 || len(matched) == 0 {
			return nil
		}
		// Extract the single matched index from the map.
		var out int
		for out = range matched {
		}
		return out
	})
}

func (p *posPredicate) computeIdentifier(value *Value) interface{} {
	return int(p.pos)
}

func (p *exprPredicate) computeIdentifier(value *Value) interface{} {
	return value.Perform(func(arr *Array) interface{} {
		if p.nodeID.identifier == "." {
			//only leaf-lists can be referenced this way
			ret := []int{}
			arr.Range(func(idx int, value *Value) {
				if value.RFC7951String() == p.value {
					ret = append(ret, idx)
				}
			})
			if len(ret) == 1 {
				return ret[0]
			}
			return ret
		}
		//only lists can be referenced this way
		ret := []int{}
		arr.Range(func(idx int, value *Value) {
			value, found := p.nodeID.Find(value)
			if found && value != nil &&
				value.RFC7951String() == p.value {
				ret = append(ret, idx)
			}
		})
		if len(ret) == 1 {
			return ret[0]
		}
		return ret
	})
}

func (id *nodeID) computeIdentifierDefault(v *Value) interface{} {
	ident := id.computeIdentifier(v)
	if ident == nil {
		return id.prefix + ":" + id.identifier
	}
	return ident
}

func (p *predicates) computeIdentifierDefault(v *Value) interface{} {
	id := p.computeIdentifier(v)
	if id == nil {
		return v.Perform(func(a *Array) int {
			// Append by default
			return a.Length()
		}, func(_ interface{}) int {
			return 0
		})
	}
	return id
}

func (p *posPredicate) computeIdentifierDefault(v *Value) interface{} {
	id := p.computeIdentifier(v)
	if id == nil {
		return 0
	}
	return id
}
func (p *exprPredicate) computeIdentifierDefault(v *Value) interface{} {
	id := p.computeIdentifier(v)
	if id == nil {
		return 0
	}
	return id
}

type matchModifier interface {
	modifyMatchCriteria(v *Value) *Value
}

func (p *predicates) modifyMatchCriteria(v *Value) *Value {
	for _, pred := range p.preds {
		mm, isMatchModifier := pred.instanceIDSelector.(matchModifier)
		if isMatchModifier {
			v = mm.modifyMatchCriteria(v)
		}
	}
	return v
}

func (p *exprPredicate) modifyMatchCriteria(v *Value) *Value {
	if p.nodeID.identifier == "." {
		return v
	}
	return v.Perform(func(o *Object) *Value {
		return ValueNew(o.Assoc(p.nodeID.identifier, p.value))
	}).(*Value)
}

type nodeCreator interface {
	createNode() *Value
}

func (id *nodeID) createNode() *Value {
	return ValueNew(ObjectNew())
}

func (p *predicates) createNode() *Value {
	return ValueNew(ArrayNew())
}
