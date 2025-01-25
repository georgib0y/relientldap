package domain

type EqualityRule int
type OrderingRule int
type SubstringRule int
type UsageType int

const (
	UserApplications UsageType = iota
	DirectoryOperations
	DistributedOperation
	DSAOperatoin
)

type Attribute struct {
	Numericoid                       OID
	Names                            map[string]bool
	Desc                             string
	Obsolete                         bool
	SupOids                          map[OID]bool
	EqRule                           EqualityRule
	OrdRule                          OrderingRule
	SubStrRule                       SubstringRule
	Syntax                           string
	SingleVal, collective, noUserMod bool
	Usage                            UsageType
	Extensions                       string
}

