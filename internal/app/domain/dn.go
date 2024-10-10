package domain

import "strings"

type AttrList map[OID]map[string]bool

type (
	ID  uint64
	OID string
)

type AVA struct {
	Oid OID
	Val string
}

func (a AVA) String() string {
	return string(a.Oid) + "=" + a.Val
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

func (d DN) ParentDN() DN {
	return d[1:]
}
