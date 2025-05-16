package schema

import (
	"fmt"
)

type UsageType int

const (
	UserApplications UsageType = iota
	DirectoryOperations
	DistributedOperation
	DSAOperatoin
)

func NewUsage(usage string) (UsageType, error) {
	switch usage {
	case "userApplications":
		return UserApplications, nil
	case "directoryOperation":
		return DirectoryOperations, nil
	case "distributedOperation":
		return DistributedOperation, nil
	case "dSAOperation":
		return DSAOperatoin, nil
	}

	return UserApplications, fmt.Errorf("unknown usage type: %s", usage)
}

type Attribute struct {
	numericoid                       OID
	names                            map[string]struct{}
	desc                             string
	obsolete                         bool
	sup                              *Attribute
	eqRule, ordRule, subStrRule      MatchingRule
	syntax                           OID
	syntaxLen                        int // max length the value can contain
	singleVal, collective, noUserMod bool
	usage                            UsageType
	// TODO extensions
	// extensions                       string
}

type AttributeBuilder struct {
	a      Attribute
	supOid OID
}

func NewAttributeBuilder() *AttributeBuilder {
	return &AttributeBuilder{
		a: Attribute{
			names:  map[string]struct{}{},
			eqRule: &UnspecifiedMatchingRule{},
			// TODO ord and substr rule?
		},
	}
}

func (b *AttributeBuilder) SetOid(numericoid OID) *AttributeBuilder {
	b.a.numericoid = numericoid
	return b
}

func (b *AttributeBuilder) AddNames(name ...string) *AttributeBuilder {
	for _, n := range name {
		b.a.names[n] = struct{}{}
	}
	return b
}

func (b *AttributeBuilder) SetDesc(desc string) *AttributeBuilder {
	b.a.desc = desc
	return b
}

func (b *AttributeBuilder) SetObsolete() *AttributeBuilder {
	b.a.obsolete = true
	return b
}

func (b *AttributeBuilder) SetSupOid(supOid OID) *AttributeBuilder {
	b.supOid = supOid
	return b
}

func (b *AttributeBuilder) SetSup(sup *Attribute) *AttributeBuilder {
	b.a.sup = sup
	return b
}

func (b *AttributeBuilder) SetEqRule(rule MatchingRule) *AttributeBuilder {
	b.a.eqRule = rule
	return b
}

func (b *AttributeBuilder) SetOrdRule(rule MatchingRule) *AttributeBuilder {
	b.a.ordRule = rule
	return b
}

func (b *AttributeBuilder) SetSubStrRule(rule MatchingRule) *AttributeBuilder {
	b.a.subStrRule = rule
	return b
}

func (b *AttributeBuilder) SetSyntax(syntax OID, len int) *AttributeBuilder {
	b.a.syntax = syntax
	b.a.syntaxLen = len
	return b
}

func (b *AttributeBuilder) SetSyntaxLength(len int) *AttributeBuilder {
	b.a.syntaxLen = len
	return b
}

func (b *AttributeBuilder) SetSingleVal() *AttributeBuilder {
	b.a.singleVal = true
	return b
}

func (b *AttributeBuilder) SetCollective() *AttributeBuilder {
	b.a.collective = true
	return b
}

func (b *AttributeBuilder) SetNoUserMod() *AttributeBuilder {
	b.a.noUserMod = true
	return b
}

func (b *AttributeBuilder) SetUsage(usage UsageType) *AttributeBuilder {
	b.a.usage = usage
	return b
}

func (b *AttributeBuilder) Resolve(attrs map[OID]*Attribute) error {
	attr, ok := attrs[b.supOid]
	if !ok {
		return fmt.Errorf("Unknown attribute oid %s", b.supOid)
	}
	b.SetSup(attr)
	return nil
}

func (b *AttributeBuilder) Build() *Attribute {
	return &b.a
}

func (a *Attribute) Oid() OID {
	return a.numericoid
}

// func (a *Attribute) Name() string {
// 	return b.
// }

func (a *Attribute) EqRule() MatchingRule {
	return a.eqRule
}

func (a *Attribute) SingleVal() bool {
	return a.singleVal
}
