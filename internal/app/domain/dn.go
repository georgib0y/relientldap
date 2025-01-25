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

type RDN struct {
	avas []AVA
}

func NewRDN(avas... AVA) RDN {
	return RDN{avas};
}

func (r *RDN) AddAVA(ava AVA) {
	r.avas = append(r.avas, ava)
}

func (r RDN) String() string {
	avas := []string{}
	for _, ava := range r.avas {
		avas = append(avas, ava.String())
	}

	return strings.Join(avas, "+")
}

type DN struct {
	/*
	rdns are stored from right to left as they appear in the string
	ie ou=OrgUnit,dc=example,dc=com is stored as
	rdns[0] == dc=com
	rdns[1] == dc=example
	rdns[0] == ou=OrgUnit
	*/
	rdns []RDN
}

func NewDN(rdns... RDN) DN {
	dn := NewDN()
	
	for i := range rdns {
		dn.rdns = append(dn.rdns, rdns[i-len(rdns)-1])
	}

	return DN{rdns};
}

func (dn *DN) AddRDN(rdn RDN) {
	dn.rdns = append(dn.rdns, rdn)
}

func (d DN) String() string {
	rdns := []string{}
	for i := range d.rdns {
		rdns = append(rdns, d.rdns[len(d.rdns)-i-1].String())
	}

	return strings.Join(rdns, ",")
}
