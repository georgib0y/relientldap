package ldif

import (
	"fmt"
	"io"
	"slices"

	d "github.com/georgib0y/relientldap/internal/domain"
)

func ParseObjectClasses(r io.Reader, attributes map[d.OID]*d.Attribute) (map[d.OID]*d.ObjectClass, error) {
	tokeniser, err := NewTokeniser(r)
	if err != nil {
		return nil, err
	}

	ldifs := []*LdifObjectClass{}
	for tokeniser.HasNext() {
		subt, err := tokeniser.ParenSubTokeniser()
		if err != nil {
			return nil, err
		}
		l, err := NewLdifObjectClass(subt)
		if err != nil {
			return nil, err
		}
		ldifs = append(ldifs, l)
	}

	resolver := NewLdifOcResolver(ldifs, attributes)
	return resolver.Resolve()
}

type LdifOcResolver struct {
	ldif  []*LdifObjectClass
	attrs map[d.OID]*d.Attribute
	ocs   map[d.OID]*d.ObjectClass
}

func NewLdifOcResolver(ldifs []*LdifObjectClass, attrs map[d.OID]*d.Attribute) *LdifOcResolver {
	return &LdifOcResolver{
		ldif:  ldifs,
		attrs: attrs,
		ocs:   map[d.OID]*d.ObjectClass{},
	}
}

func (r *LdifOcResolver) Resolve() (map[d.OID]*d.ObjectClass, error) {
	for _, l := range r.ldif {
		if err := l.Build(r); err != nil {
			return nil, err
		}
	}
	return r.ocs, nil
}

func (r *LdifOcResolver) supByOid(o d.OID) *d.ObjectClass {
	sup, ok := r.ocs[o]
	if ok {
		return sup
	}

	sup = new(d.ObjectClass)
	r.ocs[o] = sup
	return sup
}

func (r *LdifOcResolver) GetSup(nameOrOid string) (*d.ObjectClass, error) {
	for _, o := range r.ldif {
		if o.numericoid == nameOrOid {
			return r.supByOid(d.OID(o.numericoid)), nil
		}
		for _, n := range o.names {
			if nameOrOid == n {
				return r.supByOid(d.OID(o.numericoid)), nil
			}
		}
	}

	return nil, fmt.Errorf("could not find objclass sup %q", nameOrOid)
}

func (r *LdifOcResolver) GetAttr(nameOrOid string) (*d.Attribute, error) {
	for _, a := range r.attrs {
		if a.Oid() == d.OID(nameOrOid) {
			return a, nil
		}

		if a.HasName(nameOrOid) {
			return a, nil
		}
	}

	return nil, fmt.Errorf("unknown attribute %q", nameOrOid)
}

func (r *LdifOcResolver) PutObjClass(oc *d.ObjectClass) {
	if o, ok := r.ocs[oc.Oid()]; ok {
		*o = *oc
	} else {
		r.ocs[oc.Oid()] = oc
	}
}

type LdifObjectClass struct {
	numericoid string
	names      []string
	desc       string
	obsolete   bool
	sups       []string
	kind       string
	musts      []string
	mays       []string
}

func NewLdifObjectClass(t *Tokeniser) (*LdifObjectClass, error) {
	oc := &LdifObjectClass{
		names: []string{},
		sups:  []string{},
		musts: []string{},
		mays:  []string{},
	}

	if err := oc.setNumericoid(t); err != nil {
		return nil, err
	}

	keywords := []string{
		"NAME",
		"DESC",
		"OBSOLETE",
		"SUP",
		"ABSTRACT",
		"STRUCTURAL",
		"AUXILIARY",
		"MUST",
		"MAY",
	}

	for len(keywords) > 0 {
		keyword, ok := t.Next()
		if !ok {
			return oc, nil
		}

		if keyword.tokenType != KEYWORD {
			return oc, fmt.Errorf("expected KEYWORD, got %s (%s)", keyword.tokenType, keyword.val)
		}

		idx := slices.Index(keywords, keyword.val)
		if idx == -1 {
			return nil, fmt.Errorf("unkown keyword or unexpected poistion %q", keyword.val)
		}
		keywords = keywords[idx+1:]

		var err error
		switch keyword.val {
		case "NAME":
			err = oc.setName(t)
		case "DESC":
			err = oc.setDesc(t)
		case "OBSOLETE":
			err = oc.setObsolete()
		case "SUP":
			err = oc.setSup(t)
		case "ABSTRACT":
			err = oc.setKind(keyword.val)
		case "STRUCTURAL":
			err = oc.setKind(keyword.val)
		case "AUXILIARY":
			err = oc.setKind(keyword.val)
		case "MUST":
			err = oc.setMust(t)
		case "MAY":
			err = oc.setMay(t)
		}

		if err != nil {
			return nil, err
		}
	}

	return oc, nil
}

func (o *LdifObjectClass) Build(r *LdifOcResolver) error {
	b := d.NewObjectClassBuilder()
	b.SetOid(d.OID(o.numericoid)).
		AddName(o.names...).
		SetDesc(o.desc).
		SetObsolete(o.obsolete)

	if o.kind != "" {
		kind, err := d.NewKind(o.kind)
		if err != nil {
			return err
		}
		b.SetKind(kind)
	}

	for _, sup := range o.sups {
		if sup == "top" || sup == "2.5.6.0" {
			b.AddSup(d.TopObjectClass)
			continue
		}

		s, err := r.GetSup(sup)
		if err != nil {
			return err
		}
		b.AddSup(s)
	}

	for _, must := range o.musts {
		m, err := r.GetAttr(must)
		if err != nil {
			return err
		}
		b.AddMustAttr(m)
	}

	for _, may := range o.mays {
		m, err := r.GetAttr(may)
		if err != nil {
			return err
		}
		b.AddMayAttr(m)
	}

	r.PutObjClass(b.Build())
	return nil
}

func (o *LdifObjectClass) setNumericoid(t *Tokeniser) error {
	numericoid, err := t.NextNumericoid()
	if err != nil {
		return err
	}
	o.numericoid = stripQuotes(numericoid.val)
	return nil
}

func (o *LdifObjectClass) setName(t *Tokeniser) error {
	tokens, err := t.NextQdescrs()
	if err != nil {
		return err
	}

	for _, token := range tokens {
		o.names = append(o.names, stripQuotes(token.val))
	}
	return nil
}

func (o *LdifObjectClass) setDesc(t *Tokeniser) error {
	desc, err := t.NextQdstring()
	if err != nil {
		return err
	}
	o.desc = stripQuotes(desc.val)
	return nil
}

func (o *LdifObjectClass) setObsolete() error {
	o.obsolete = true
	return nil
}

func (o *LdifObjectClass) setSup(t *Tokeniser) error {
	sups, err := t.NextOids()
	if err != nil {
		return err
	}

	oids := []string{}
	for _, s := range sups {
		oids = append(oids, stripQuotes(s.val))
	}

	o.sups = oids
	return nil
}

func (o *LdifObjectClass) setKind(kind string) error {
	o.kind = stripQuotes(kind)
	return nil
}

func (o *LdifObjectClass) setMust(t *Tokeniser) error {
	musts, err := t.NextOids()
	if err != nil {
		return err
	}

	oids := []string{}
	for _, s := range musts {
		oids = append(oids, stripQuotes(s.val))
	}

	o.musts = oids
	return nil
}

func (o *LdifObjectClass) setMay(t *Tokeniser) error {
	mays, err := t.NextOids()
	if err != nil {
		return err
	}

	oids := []string{}
	for _, s := range mays {
		oids = append(oids, stripQuotes(s.val))
	}

	o.mays = oids
	return nil
}
