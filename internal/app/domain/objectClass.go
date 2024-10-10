package domain

type ObjectClassKind int

const (
	Abstract ObjectClassKind = iota
	Structural
	Auxilary
)

type ObjectClass struct {
	Numericoid          OID
	Names               map[string]bool
	Desc                string
	Obsolete            bool
	SupOids             map[OID]bool
	Kind                ObjectClassKind
	MustAttrs, MayAttrs map[OID]bool
}
