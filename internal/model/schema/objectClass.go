package schema

import (
	"fmt"
	"strings"

	"github.com/georgib0y/relientldap/internal/model/dit"
)

type ObjectClassKind int

const (
	Abstract ObjectClassKind = iota
	Structural
	Auxilary
)

func (k ObjectClassKind) String() string {
	switch k {
	case Abstract:
		return "ABSTRACT"
	case Structural:
		return "STRUCTURAL"
	case Auxilary:
		return "AUXILARY"
	}

	return fmt.Sprintf("unknown (%d)", k)
}

type ObjectClass struct {
	numericoid          dit.OID
	names               map[string]bool
	desc                string
	obsolete            bool
	supOids             map[dit.OID]bool
	kind                ObjectClassKind
	mustAttrs, mayAttrs map[dit.OID]bool
}

type ObjectClassOption func(*ObjectClass)

func ObjClassWithOid(oid dit.OID) ObjectClassOption {
	return func(oc *ObjectClass) {
		oc.numericoid = oid
	}
}

func ObjClassWithName(names ...string) ObjectClassOption {
	return func(oc *ObjectClass) {
		for _, n := range names {
			oc.names[n] = true
		}
	}
}

func ObjClassWithDesc(desc string) ObjectClassOption {
	return func(oc *ObjectClass) {
		oc.desc = desc
	}
}

func ObjClassWithObsolete() ObjectClassOption {
	return func(oc *ObjectClass) {
		oc.obsolete = true
	}
}

func ObjClassWithSupOid(sup ...dit.OID) ObjectClassOption {
	return func(oc *ObjectClass) {
		for _, s := range sup {
			oc.supOids[s] = true
		}
	}
}

func ObjClassWithKind(kind ObjectClassKind) ObjectClassOption {
	return func(oc *ObjectClass) {
		oc.kind = kind
	}
}

func ObjClassWithMustAttr(attr ...dit.OID) ObjectClassOption {
	return func(oc *ObjectClass) {
		for _, a := range attr {
			oc.mustAttrs[a] = true
		}
	}
}

func ObjClassWithMayAttr(attr ...dit.OID) ObjectClassOption {
	return func(oc *ObjectClass) {
		for _, a := range attr {
			oc.mayAttrs[a] = true
		}
	}
}

func NewObjectClass(options ...ObjectClassOption) ObjectClass {
	oc := ObjectClass{
		names:     map[string]bool{},
		supOids:   map[dit.OID]bool{},
		mustAttrs: map[dit.OID]bool{},
		mayAttrs:  map[dit.OID]bool{},
	}

	for _, o := range options {
		o(&oc)
	}

	return oc
}

func (o ObjectClass) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Numericoid: %s\n", string(o.numericoid))
	sb.WriteString("Names:")
	for n := range o.names {
		sb.WriteString(" " + n)
	}
	fmt.Fprintf(&sb, "\nDesc: %s\n", o.desc)
	fmt.Fprintf(&sb, "Obsolete: %t\n", o.obsolete)
	sb.WriteString("Sup Oids:")
	for s := range o.supOids {
		sb.WriteString(" " + string(s))
	}
	fmt.Fprintf(&sb, "\nKind: %s\n", o.kind)

	sb.WriteString("Must:")
	for a := range o.mustAttrs {
		sb.WriteString(" " + string(a))
	}

	sb.WriteString("\nMay:")
	for a := range o.mayAttrs {
		sb.WriteString(" " + string(a))
	}

	return sb.String()
}

// TODO move this somewhere more useful
func mapSetsEq[K comparable](m1, m2 map[K]bool) bool {
	if len(m1) != len(m2) {
		return false
	}

	for k := range m1 {
		if _, ok := m2[k]; !ok {
			return false
		}
	}

	return true
}

func ObjectClassesAreEqual(o1, o2 ObjectClass) error {
	switch {
	case o1.numericoid != o2.numericoid:
		return fmt.Errorf("numericoids dont match")
	case !mapSetsEq(o1.names, o2.names):
		return fmt.Errorf("names dont match")
	case o1.desc != o2.desc:
		return fmt.Errorf("descs dont match")
	case o1.obsolete != o2.obsolete:
		return fmt.Errorf("obsoletes dont match")
	case !mapSetsEq(o1.supOids, o2.supOids):
		return fmt.Errorf("supoids dont match")
	case o1.kind != o2.kind:
		return fmt.Errorf("kinds dont match")
	case !mapSetsEq(o1.mustAttrs, o2.mustAttrs):
		return fmt.Errorf("musts dont match")
	case !mapSetsEq(o1.mayAttrs, o2.mayAttrs):
		return fmt.Errorf("mays dont match")
	default:
		return nil
	}
}
