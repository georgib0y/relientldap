package schema

import (
	"github.com/georgib0y/relientldap/internal/app/domain/dit"
)

type ObjectClassKind int

const (
	Abstract ObjectClassKind = iota
	Structural
	Auxilary
)

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

func WithObsolete() ObjectClassOption {
	return func(oc *ObjectClass) {
		oc.obsolete = true
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
