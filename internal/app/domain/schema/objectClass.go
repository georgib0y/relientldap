package schema

import "github.com/georgib0y/relientldap/internal/app/domain/dit"

type ObjectClassKind int

const (
	Abstract ObjectClassKind = iota
	Structural
	Auxilary
)

type ObjectClass struct {
	Numericoid          dit.OID
	Names               map[string]bool
	Desc                string
	Obsolete            bool
	SupOids             map[dit.OID]bool
	Kind                ObjectClassKind
	MustAttrs, MayAttrs map[dit.OID]bool
}
