package domain

import (
	"slices"
	"strings"
)

type (
	ID uint64
)

// type AVA struct {
// 	attr *schema.Attribute
// 	Val  string
// }

// func (a AVA) String() string {
// 	return string(a.attr.Oid()) + "=" + a.Val
// }

type RDNOption func(*RDN)

func WithAVA(attr *Attribute, val string) RDNOption {
	return func(r *RDN) {
		r.avas[attr] = val
	}
}

type RDN struct {
	avas map[*Attribute]string
}

func NewRDN(options ...RDNOption) RDN {
	r := RDN{map[*Attribute]string{}}

	for _, o := range options {
		o(&r)
	}

	return r
}

func (r RDN) Clone() RDN {
	avas := map[*Attribute]string{}
	for o, a := range r.avas {
		avas[o] = a
	}

	return RDN{avas}
}

func CompareRDNs(r1, r2 *RDN) bool {
	if len(r1.avas) != len(r2.avas) {
		return false
	}

	for attr, val1 := range r1.avas {
		val2, ok := r2.avas[attr]
		if !ok {
			return false
		}

		eq, ok := attr.EqRule()
		if !ok {
			logger.Printf("attribute %s does not have an eq rule", attr)
			return false
		}

		if ok, err := eq.Match(val1, val2); !ok || err != nil {
			return false
		}
	}

	return true
}

func (r RDN) String() string {
	avas := []string{}
	for attr, val := range r.avas {
		ava := attr.Name() + "=" + val
		avas = append(avas, ava)
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

type DnBuilder struct {
	dn DN
}

func NewDnBuilder() *DnBuilder {
	return &DnBuilder{dn: DN{rdns: []RDN{}}}
}

// TODO does the of context strings make sense?
func (b *DnBuilder) AddNamingContext(dcAttr *Attribute, context ...string) *DnBuilder {
	for _, dc := range context {
		b.AddAvaAsRdn(dcAttr, dc)
	}
	return b
}

func (b *DnBuilder) AddAvaAsRdn(attr *Attribute, val string) *DnBuilder {
	b.dn.rdns = append(b.dn.rdns, NewRDN(WithAVA(attr, val)))
	return b
}

func (b *DnBuilder) AddAvaToCurrentRdn(attr *Attribute, val string) *DnBuilder {
	if len(b.dn.rdns) == 0 {
		return b.AddAvaAsRdn(attr, val)
	}

	b.dn.rdns[len(b.dn.rdns)-1].avas[attr] = val
	return b
}

func (b *DnBuilder) Build() DN {
	return b.dn
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
		if !CompareRDNs(&dn1.rdns[i], &dn2.rdns[i]) {
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
func (dn *DN) GetRDN() *RDN {
	return &dn.rdns[len(dn.rdns)-1]
}

func (dn DN) GetParentDN() DN {
	return DN{dn.rdns[:len(dn.rdns)-1]}
}

func (d *DN) String() string {
	if d == nil {
		return ""
	}
	rdns := []string{}
	for i := range d.rdns {
		rdns = append(rdns, d.rdns[len(d.rdns)-i-1].String())
	}

	return strings.Join(rdns, ",")
}

func attrValFromStr(schema *Schema, s string) (*Attribute, string, error) {
	spl := strings.Split(s, "=")
	// TODO could be wrong
	if len(spl) != 2 {
		// TODO should technically be providing a matched dn here but to hard
		return nil, "", NewLdapError(InvalidDnSyntax, nil, "malformed ava: %s", s)
	}

	attr, ok := schema.FindAttribute(strings.TrimSpace(spl[0]))
	if !ok {
		// return nil, "", fmt.Errorf("unknown attribute %q", strings.TrimSpace(spl[0]))
		return nil, "", NewLdapError(UndefinedAttributeType, nil, "unknown attribute %q", strings.TrimSpace(spl[0]))
	}

	return attr, spl[1], nil
}

// TODO this is definitely not a complete DN parser, though probs good enough for now
func NormaliseDN(schema *Schema, s string) (DN, error) {
	b := NewDnBuilder()
	rdns := strings.Split(s, ",")
	slices.Reverse(rdns)

	for _, spl := range rdns {
		avas := strings.Split(spl, "+")
		a, v, err := attrValFromStr(schema, avas[0])
		if err != nil {
			return DN{}, err
		}
		b.AddAvaAsRdn(a, v)
		for _, ava := range avas[1:] {
			a, v, err := attrValFromStr(schema, ava)
			if err != nil {
				return DN{}, err
			}
			b.AddAvaToCurrentRdn(a, v)
		}
	}

	return b.Build(), nil
}
