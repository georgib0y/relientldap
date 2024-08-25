package main

import "strings"

type (
	ID  uint64
	OID string
)

type AttrList map[OID]map[string]bool

type AVA struct {
	oid OID
	val string
}

func (a AVA) String() string {
	return string(a.oid) + "=" + a.val
}

type RDN []AVA

func (r RDN) String() string {
	avas := []string{}
	for _, ava := range r {
		avas = append(avas, ava.String())
	}

	return strings.Join(avas, "+")
}

type DN []RDN

func (d DN) String() string {
	rdns := []string{}
	for _, rdn := range d {
		rdns = append(rdns, rdn.String())
	}

	return strings.Join(rdns, ",")
}

func (d DN) parentDN() DN {
	return d[1:]
}

type Entry struct {
	id, parent ID
	children   map[ID]bool
	objClasses map[OID]bool
	attrs      map[OID]map[string]bool
}

type ObjectClassKind int

const (
	Abstract ObjectClassKind = iota
	Structural
	Auxilary
)

type ObjectClass struct {
	numericoid          OID
	names               map[string]bool
	desc                string
	obsolete            bool
	supOids             map[OID]bool
	kind                ObjectClassKind
	mustAttrs, mayAttrs map[OID]bool
}

type EqualityRule int
type OrderingRule int
type SubstringRule int
type UsageType int

const (
	UserApplications UsageType = iota
	DirectoryOperations
	DistributedOperation
	DSAOperatoin
)

type Attribute struct {
	numericoid                       OID
	names                            map[string]bool
	desc                             string
	obsolete                         bool
	supOids                          map[OID]bool
	eqRule                           EqualityRule
	ordRule                          OrderingRule
	subStrRule                       SubstringRule
	syntax                           string
	singleVal, collective, noUserMod bool
	usage                            UsageType
	extensions                       string
}
