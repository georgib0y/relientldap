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

func WithOid(oid dit.OID) ObjectClassOption {
	return func(oc *ObjectClass) {
		oc.numericoid = oid
	}
}

func WithName(names ...string) ObjectClassOption {
	return func(oc *ObjectClass) {
		for _, n := range names {
			oc.names[n] = true
		}
	}
}

func WithDesc(desc string) ObjectClassOption {
	return func(oc *ObjectClass) {
		oc.desc = desc
	}
}

func WithObsolete(obsolete bool) ObjectClassOption {
	return func(oc *ObjectClass) {
		oc.obsolete = obsolete
	}
}

func WithSupOid(sup ...dit.OID) ObjectClassOption {
	return func(oc *ObjectClass) {
		for _, s := range sup {
			oc.supOids[s] = true
		}
	}
}

func WithKind(kind ObjectClassKind) ObjectClassOption {
	return func(oc *ObjectClass) {
		oc.kind = kind
	}
}

func WithMustAttr(attr ...dit.OID) ObjectClassOption {
	return func(oc *ObjectClass) {
		for _, a := range attr {
			oc.mustAttrs[a] = true
		}
	}
}

func WithMayAttr(attr ...dit.OID) ObjectClassOption {
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
