package ldif

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	d "github.com/georgib0y/relientldap/internal/domain"
)

func ParseAttributes(r io.Reader) (map[d.OID]*d.Attribute, error) {
	tokeniser, err := NewTokeniser(r)
	if err != nil {
		return nil, err
	}

	ldifs := []*LdifAttribute{}
	for tokeniser.HasNext() {
		subt, err := tokeniser.ParenSubTokeniser()
		if err != nil {
			return nil, err
		}
		l, err := NewLdifAttribute(subt)
		if err != nil {
			return nil, err
		}
		ldifs = append(ldifs, l)
	}

	resolver := NewLdifAttrResolver(ldifs)
	return resolver.Resolve()
}

type LdifAttrResolver struct {
	ldif  []*LdifAttribute
	attrs map[d.OID]*d.Attribute
}

func NewLdifAttrResolver(ldifs []*LdifAttribute) *LdifAttrResolver {
	return &LdifAttrResolver{ldifs, map[d.OID]*d.Attribute{}}
}

func (r *LdifAttrResolver) Resolve() (map[d.OID]*d.Attribute, error) {
	for _, l := range r.ldif {
		if err := l.Build(r); err != nil {
			return nil, err
		}
	}

	return r.attrs, nil
}

func (r *LdifAttrResolver) supByOid(o d.OID) *d.Attribute {
	sup, ok := r.attrs[o]
	if ok {
		return sup
	}

	sup = new(d.Attribute)
	r.attrs[o] = sup
	return sup
}

func (r *LdifAttrResolver) GetSup(nameOrOid string) (*d.Attribute, error) {
	for _, a := range r.ldif {
		if a.numericoid == nameOrOid {
			return r.supByOid(d.OID(a.numericoid)), nil
		}
		for _, n := range a.names {
			if nameOrOid == n {
				return r.supByOid(d.OID(a.numericoid)), nil
			}
		}
	}

	return nil, fmt.Errorf("could not find attr sup %q", nameOrOid)
}

func (r *LdifAttrResolver) PutAttr(attr *d.Attribute) {
	if a, ok := r.attrs[attr.Oid()]; ok {
		*a = *attr
	} else {
		r.attrs[attr.Oid()] = attr
	}
}

type LdifAttribute struct {
	numericoid      string
	names           []string
	desc            string
	obsolete        bool
	sup             string
	eq, ord, substr string
	syntax          string
	syntaxLen       int
	singleVal       bool
	collective      bool
	noUserMod       bool
	usage           string
	// TODO extensions
}

func NewLdifAttribute(t *Tokeniser) (*LdifAttribute, error) {
	attr := &LdifAttribute{
		names: []string{},
	}

	if err := attr.setNumericoid(t); err != nil {
		return nil, err
	}

	keywords := []string{
		"NAME",
		"DESC",
		"OBSOLETE",
		"SUP",
		"EQUALITY",
		"ORDERING",
		"SUBSTR",
		"SYNTAX",
		"SINGLE-VALUE",
		"COLLECTIVE",
		"NO-USER-MODIFICATION",
		"USAGE",
	}

	for len(keywords) > 0 {
		keyword, ok := t.Next()
		if !ok {
			return attr, nil
		}

		if keyword.tokenType != KEYWORD {
			return nil, fmt.Errorf("expected KEYWORD got %s (%s)", keyword.tokenType, keyword.val)
		}

		idx := slices.Index(keywords, keyword.val)
		if idx == -1 {
			return nil, fmt.Errorf("unknown keyword or in unexpected position %q", keyword.val)
		}
		keywords = keywords[idx+1:]

		var err error
		switch keyword.val {
		case "NAME":
			err = attr.setName(t)
		case "DESC":
			err = attr.setDesc(t)
		case "OBSOLETE":
			err = attr.setObsolete()
		case "SUP":
			err = attr.setSup(t)
		case "EQUALITY":
			err = attr.setEq(t)
		case "ORDERING":
			err = attr.setOrdering(t)
		case "SUBSTR":
			err = attr.setSubstr(t)
		case "SYNTAX":
			err = attr.setSyntax(t)
		case "SINGLE-VALUE":
			err = attr.setSingleVal()
		case "COLLECTIVE":
			err = attr.setCollective()
		case "NO-USER-MODIFICATION":
			err = attr.setNoUserMod()
		case "USAGE":
			err = attr.setUsage(t)
		default:
			return nil, fmt.Errorf("unknown attribute keyword %q", keyword.val)
		}

		if err != nil {
			return nil, err
		}
	}

	return attr, nil
}

// builds and places this attribute into the attribute map, uses the others slice to get references to sups
func (a *LdifAttribute) Build(r *LdifAttrResolver) error {
	b := d.NewAttributeBuilder()
	b.SetOid(d.OID(a.numericoid)).
		AddNames(a.names...).
		SetDesc(a.desc).
		SetObsolete(a.obsolete).
		SetSyntax(d.OID(a.syntax), a.syntaxLen).
		SetSingleVal(a.singleVal).
		SetCollective(a.collective).
		SetNoUserMod(a.noUserMod)

	if a.sup != "" {
		sup, err := r.GetSup(a.sup)
		if err != nil {
			return err
		}
		b.SetSup(sup)
	}

	if a.eq != "" {
		eq, err := d.GetMatchingRule(a.eq)
		if err != nil {
			return err
		}
		b.SetEqRule(eq)
	}

	if a.ord != "" {
		ord, err := d.GetMatchingRule(a.ord)
		if err != nil {
			return err
		}
		b.SetOrdRule(ord)
	}

	if a.substr != "" {
		sub, err := d.GetMatchingRule(a.substr)
		if err != nil {
			return err
		}
		b.SetSubStrRule(sub)
	}

	if a.usage != "" {
		usage, err := d.NewUsage(a.usage)
		if err != nil {
			return err
		}
		b.SetUsage(usage)
	}

	r.PutAttr(b.Build())
	return nil
}

func (a *LdifAttribute) setNumericoid(t *Tokeniser) error {
	numericoid, err := t.NextNumericoid()
	if err != nil {
		return err
	}
	a.numericoid = numericoid.val
	return nil
}

func (a *LdifAttribute) setName(t *Tokeniser) error {
	tokens, err := t.NextQdescrs()
	if err != nil {
		return err
	}

	for _, token := range tokens {
		a.names = append(a.names, stripQuotes(token.val))
	}
	return nil
}

func (a *LdifAttribute) setDesc(t *Tokeniser) error {
	desc, err := t.NextQdstring()
	if err != nil {
		return err
	}
	a.desc = stripQuotes(desc.val)
	return nil
}

func (a *LdifAttribute) setObsolete() error {
	a.obsolete = true
	return nil
}

func (a *LdifAttribute) setSup(t *Tokeniser) error {
	sup, err := t.NextOid()
	if err != nil {
		return err
	}
	a.sup = stripQuotes(sup.val)
	return nil
}

func (a *LdifAttribute) setEq(t *Tokeniser) error {
	eq, err := t.NextOid()
	if err != nil {
		return err
	}
	a.eq = stripQuotes(eq.val)
	return nil
}

func (a *LdifAttribute) setOrdering(t *Tokeniser) error {
	ord, err := t.NextOid()
	if err != nil {
		return err
	}
	a.ord = stripQuotes(ord.val)
	return nil
}

func (a *LdifAttribute) setSubstr(t *Tokeniser) error {
	sub, err := t.NextOid()
	if err != nil {
		return err
	}
	a.substr = stripQuotes(sub.val)
	return nil
}

func (a *LdifAttribute) setSyntax(t *Tokeniser) error {
	stx, err := t.NextNoidlen()
	if err != nil {
		return err
	}
	// TODO handle if noidlen curly brackets
	if stx.tokenType == NUMERICOID {
		a.syntax = stripQuotes(stx.val)
		return nil
	}

	spl := strings.Split(stx.val, "{")
	oid := spl[0]
	len, err := strconv.Atoi(spl[1][:len(spl[1])-1])
	if err != nil {
		return err
	}

	a.syntax = oid
	a.syntaxLen = len
	return nil
}

func (a *LdifAttribute) setSingleVal() error {
	a.singleVal = true
	return nil
}

func (a *LdifAttribute) setCollective() error {
	a.collective = true
	return nil
}

func (a *LdifAttribute) setNoUserMod() error {
	a.noUserMod = true
	return nil
}

func (a *LdifAttribute) setUsage(t *Tokeniser) error {
	usage, err := t.NextDescr()
	if err != nil {
		return err
	}
	a.usage = stripQuotes(usage.val)
	return nil
}
