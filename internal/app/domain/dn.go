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

type RDNOption func(*RDN)

func WithAVA(oid OID, val string) RDNOption {
	return func(r *RDN) {
		r.AddAVA(AVA{oid, val})
	}
}

type RDN struct {
	avas map[AVA]bool
}

func NewRDN(options ...RDNOption) RDN {
	r := RDN{map[AVA]bool{}}

	for _, o := range options {
		o(&r)
	}

	return r
}

func (r RDN) Clone() RDN {
	avas := map[AVA]bool{}
	for a := range r.avas {
		avas[a] = true
	}

	return RDN{avas}
}

func (r *RDN) AddAVA(ava AVA) {
	r.avas[ava] = true
}

// TODO better name?
func CompareRDNs(r1, r2 RDN) bool {
	if len(r1.avas) != len(r2.avas) {
		return false
	}

	for ava := range r1.avas {
		if _, ok := r2.avas[ava]; !ok {
			return false
		}
	}

	return true
}

func (r RDN) String() string {
	avas := []string{}
	for ava := range r.avas {
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
		rdns[2] == ou=OrgUnit
	*/
	rdns []RDN
}

type DNOption func(*DN)

func WithRdnAppended(rdn RDN) DNOption {
	return func(dn *DN) {
		dn.rdns = append(dn.rdns, rdn)
	}
}

func WithRDN(rdn RDN) DNOption {
	return func(dn *DN) {
		dn.rdns = append([]RDN{rdn}, dn.rdns...)
	}
}

func WithRdnAva(oid OID, val string) DNOption {
	return WithRDN(NewRDN(WithAVA(oid, val)))
}

func NewDN(options ...DNOption) DN {
	dn := DN{}

	for _, o := range options {
		o(&dn)
	}

	return dn
}

func (dn DN) Clone() DN {
	rdns := []RDN{}
	for _, r := range dn.rdns {
		rdns = append(rdns, r.Clone())
	}

	return DN{rdns}
}

func CompareDNs(dn1, dn2 DN) bool {
	if len(dn1.rdns) != len(dn2.rdns) {
		return false
	}

	for i := range dn1.rdns {
		if !CompareRDNs(dn1.rdns[i], dn2.rdns[2]) {
			return false
		}
	}

	return true
}

func (dn *DN) AddRDN(rdn RDN) {
	dn.rdns = append(dn.rdns, rdn)
}

// Replaces the deepest rdn (the first rdn that shows when stringified) with a new rdn
// Useful for the ModifyDN request
func (dn *DN) ReplaceRDN(rdn RDN) {
	dn.rdns[len(dn.rdns)-1] = rdn
}

// Returns the deepest rdn
// TODO not sure this func is obvious enough
func (dn DN) GetRDN() RDN {
	return dn.rdns[len(dn.rdns)-1]
}

func (dn DN) GetParentDN() DN {
	return DN{dn.rdns[:len(dn.rdns)-1]}
}

func (d DN) String() string {
	rdns := []string{}
	for i := range d.rdns {
		rdns = append(rdns, d.rdns[len(d.rdns)-i-1].String())
	}

	return strings.Join(rdns, ",")
}
