package model

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/georgib0y/relientldap/internal/util"
)

var (
	TopObjectClass = NewObjectClassBuilder().
		SetOid("2.5.6.0").
		AddName("top").
		SetKind(Abstract).
		AddMustAttr(ObjectClassAttribute).
		Build()
)

type ObjectClassKind int

const (
	Abstract ObjectClassKind = iota
	Structural
	Auxiliary
)

func NewKind(k string) (ObjectClassKind, error) {
	switch k {
	case "ABSTRACT":
		return Abstract, nil
	case "STRUCTURAL":
		return Structural, nil
	case "AUXILIARY":
		return Auxiliary, nil
	default:
		return Abstract, fmt.Errorf("unknown kind %q", k)
	}
}

func (k ObjectClassKind) String() string {
	switch k {
	case Abstract:
		return "ABSTRACT"
	case Structural:
		return "STRUCTURAL"
	case Auxiliary:
		return "AUXILIARY"
	}

	return fmt.Sprintf("unknown (%d)", k)
}

type ObjectClass struct {
	numericoid          OID
	names               map[string]struct{}
	friendlyName        string
	desc                string
	obsolete            bool
	sups                map[OID]*ObjectClass
	kind                ObjectClassKind
	mustAttrs, mayAttrs map[OID]*Attribute
}

type ObjectClassBuilder struct {
	o        ObjectClass
	supNames map[string]struct{} // sup names is either a given name or oid for a sup object class
}

func NewObjectClassBuilder() *ObjectClassBuilder {
	return &ObjectClassBuilder{
		o: ObjectClass{
			names:     map[string]struct{}{},
			sups:      map[OID]*ObjectClass{},
			mustAttrs: map[OID]*Attribute{},
			mayAttrs:  map[OID]*Attribute{},
		},
		supNames: map[string]struct{}{},
	}
}

func (b *ObjectClassBuilder) SetOid(numericoid OID) *ObjectClassBuilder {
	b.o.numericoid = numericoid
	return b
}

func (b *ObjectClassBuilder) AddName(name ...string) *ObjectClassBuilder {
	for _, n := range name {
		if b.o.friendlyName == "" || len(b.o.friendlyName) > len(n) {
			b.o.friendlyName = n
		}
		b.o.names[n] = struct{}{}
	}
	return b
}

func (b *ObjectClassBuilder) SetDesc(desc string) *ObjectClassBuilder {
	b.o.desc = desc
	return b
}

func (b *ObjectClassBuilder) SetObsolete(o bool) *ObjectClassBuilder {
	b.o.obsolete = o
	return b
}

// TODO oid <> string not beautiful
func (b *ObjectClassBuilder) AddSupName(name ...OID) *ObjectClassBuilder {
	for _, n := range name {
		b.supNames[string(n)] = struct{}{}
	}
	return b
}

func (b *ObjectClassBuilder) AddSup(sup ...*ObjectClass) *ObjectClassBuilder {
	for _, s := range sup {
		b.o.sups[s.numericoid] = s
	}
	return b
}

func (b *ObjectClassBuilder) SetKind(kind ObjectClassKind) *ObjectClassBuilder {
	b.o.kind = kind
	return b
}

func (b *ObjectClassBuilder) AddMustAttr(attr ...*Attribute) *ObjectClassBuilder {
	for _, a := range attr {
		b.o.mustAttrs[a.numericoid] = a
	}
	return b
}

func (b *ObjectClassBuilder) AddMayAttr(attr ...*Attribute) *ObjectClassBuilder {
	for _, a := range attr {
		b.o.mayAttrs[a.numericoid] = a
	}
	return b
}

func (b *ObjectClassBuilder) Resolve(objClasses map[OID]*ObjectClass) error {
outter:
	for name := range b.supNames {
		sup, ok := objClasses[OID(name)]
		if ok {
			b.AddSup(sup)
			continue
		}

		for _, objClass := range objClasses {
			if _, ok = objClass.names[name]; ok {
				b.AddSup(objClass)
				continue outter
			}
		}

		return fmt.Errorf("Could not find sup object class %s", name)
	}

	return nil
}

func (b *ObjectClassBuilder) Build() *ObjectClass {
	return &b.o
}

func (o *ObjectClass) Oid() OID {
	return o.numericoid
}

func (o *ObjectClass) Name() string {
	if o.friendlyName != "" {
		return o.friendlyName
	}

	return string(o.numericoid)
}

func (o *ObjectClass) Kind() ObjectClassKind {
	return o.kind
}

func (o *ObjectClass) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Numericoid: %s\n", string(o.numericoid))
	sb.WriteString("Names:")
	for n := range o.names {
		sb.WriteString(" " + n)
	}
	fmt.Fprintf(&sb, "\nDesc: %s\n", o.desc)
	fmt.Fprintf(&sb, "Obsolete: %t\n", o.obsolete)
	sb.WriteString("Sup Oids:")
	for oid := range o.sups {
		sb.WriteString(" " + string(oid))
	}
	fmt.Fprintf(&sb, "\nKind: %s\n", o.kind)

	sb.WriteString("Must:")
	for oid := range o.mustAttrs {
		sb.WriteString(" " + string(oid))
	}

	sb.WriteString("\nMay:")
	for oid := range o.mayAttrs {
		sb.WriteString(" " + string(oid))
	}

	return sb.String()
}

func ObjectClassesAreEqual(o1, o2 *ObjectClass) error {
	switch {
	case o1.numericoid != o2.numericoid:
		return fmt.Errorf("numericoids dont match")
	case !util.CmpMapKeys(o1.names, o2.names):
		return fmt.Errorf("names dont match")
	case o1.desc != o2.desc:
		return fmt.Errorf("descs dont match")
	case o1.obsolete != o2.obsolete:
		return fmt.Errorf("obsoletes dont match")
	case !util.CmpMapKeys(o1.sups, o2.sups):
		return fmt.Errorf("supoids dont match")
	case o1.kind != o2.kind:
		return fmt.Errorf("kinds dont match")
	case !util.CmpMapKeys(o1.mustAttrs, o2.mustAttrs):
		return fmt.Errorf("musts dont match")
	case !util.CmpMapKeys(o1.mayAttrs, o2.mayAttrs):
		return fmt.Errorf("mays dont match")
	default:
		return nil
	}
}

func AllObjectClassMusts(e *Entry) map[*Attribute]struct{} {
	a := map[*Attribute]struct{}{}
	for _, must := range e.structural.mustAttrs {
		a[must] = struct{}{}
	}
	for oc := range e.auxiliary {
		for _, must := range oc.mustAttrs {
			a[must] = struct{}{}
		}
	}
	return a
}

func AllObjectClassMays(e *Entry) map[*Attribute]struct{} {
	a := map[*Attribute]struct{}{}
	for _, may := range e.structural.mayAttrs {
		a[may] = struct{}{}
	}
	for oc := range e.auxiliary {
		for _, may := range oc.mayAttrs {
			a[may] = struct{}{}
		}
	}
	return a
}

func ParseObjectClasses(r io.Reader, attributes map[OID]*Attribute) (map[OID]*ObjectClass, error) {
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
	attrs map[OID]*Attribute
	ocs   map[OID]*ObjectClass
}

func NewLdifOcResolver(ldifs []*LdifObjectClass, attrs map[OID]*Attribute) *LdifOcResolver {
	return &LdifOcResolver{
		ldif:  ldifs,
		attrs: attrs,
		ocs:   map[OID]*ObjectClass{},
	}
}

func (r *LdifOcResolver) Resolve() (map[OID]*ObjectClass, error) {
	for _, l := range r.ldif {
		if err := l.Build(r); err != nil {
			return nil, err
		}
	}
	return r.ocs, nil
}

func (r *LdifOcResolver) supByOid(o OID) *ObjectClass {
	sup, ok := r.ocs[o]
	if ok {
		return sup
	}

	sup = new(ObjectClass)
	r.ocs[o] = sup
	return sup
}

func (r *LdifOcResolver) GetSup(nameOrOid string) (*ObjectClass, error) {
	for _, o := range r.ldif {
		if o.numericoid == nameOrOid {
			return r.supByOid(OID(o.numericoid)), nil
		}
		for _, n := range o.names {
			if nameOrOid == n {
				return r.supByOid(OID(o.numericoid)), nil
			}
		}
	}

	return nil, fmt.Errorf("could not find objclass sup %q", nameOrOid)
}

func (r *LdifOcResolver) GetAttr(nameOrOid string) (*Attribute, error) {
	for _, a := range r.attrs {
		if a.Oid() == OID(nameOrOid) {
			return a, nil
		}

		if a.HasName(nameOrOid) {
			return a, nil
		}
	}

	return nil, fmt.Errorf("unknown attribute %q", nameOrOid)
}

func (r *LdifOcResolver) PutObjClass(oc *ObjectClass) {
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
	b := NewObjectClassBuilder()
	b.SetOid(OID(o.numericoid)).
		AddName(o.names...).
		SetDesc(o.desc).
		SetObsolete(o.obsolete)

	if o.kind != "" {
		kind, err := NewKind(o.kind)
		if err != nil {
			return err
		}
		b.SetKind(kind)
	}

	for _, sup := range o.sups {
		if sup == "top" || sup == "2.5.6.0" {
			b.AddSup(TopObjectClass)
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
