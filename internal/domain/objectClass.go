package model

import (
	"fmt"
	"strings"

	"github.com/georgib0y/relientldap/internal/util"
)

var TopObjectClass = NewObjectClassBuilder().
	SetOid("2.5.6.0").
	AddName("top").
	SetKind(Abstract).
	AddMustAttr(ObjectClassAttribute).
	Build()

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
