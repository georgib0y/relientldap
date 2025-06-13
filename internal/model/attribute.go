package model

import (
	"fmt"
	"strings"

	"github.com/georgib0y/relientldap/internal/util"
)

type UsageType int

const (
	UserApplications UsageType = iota
	DirectoryOperations
	DistributedOperation
	DsaOperation
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
		return DsaOperation, nil
	}

	return UserApplications, fmt.Errorf("unknown usage type: %s", usage)
}

func (u UsageType) String() string {
	switch u {
	case UserApplications:
		return "userApplications"
	case DirectoryOperations:
		return "directoryOperation"
	case DistributedOperation:
		return "distributedOperation"
	case DsaOperation:
		return "dsaOperation"
	default:
		return "unknown usage"
	}
}

type Attribute struct {
	numericoid                       OID
	names                            map[string]struct{}
	desc                             string
	obsolete                         bool
	sup                              *Attribute
	eqRule, ordRule, subStrRule      *MatchingRule
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
			names: map[string]struct{}{},
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

func (b *AttributeBuilder) SetEqRule(rule *MatchingRule) *AttributeBuilder {
	b.a.eqRule = rule
	return b
}

func (b *AttributeBuilder) SetOrdRule(rule *MatchingRule) *AttributeBuilder {
	b.a.ordRule = rule
	return b
}

func (b *AttributeBuilder) SetSubStrRule(rule *MatchingRule) *AttributeBuilder {
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
	if b.supOid == "" {
		return nil
	}

	attr, ok := attrs[b.supOid]
	if ok {
		b.SetSup(attr)
		return nil
	}

	// TODO could speed this up, also could be oid but that case is not handled yet
	for _, attr := range attrs {
		if attr.HasName(string(b.supOid)) {
			b.SetSup(attr)
			return nil
		}
	}

	return fmt.Errorf("Unknown attribute oid %s", b.supOid)
}

func (b *AttributeBuilder) Build() *Attribute {
	return &b.a
}

func (a *Attribute) Oid() OID {
	return a.numericoid
}

func (a *Attribute) HasName(name string) bool {
	_, ok := a.names[name]
	return ok
}

func (a *Attribute) EqRule() *MatchingRule {
	return a.eqRule
}

func (a *Attribute) SingleVal() bool {
	return a.singleVal
}

func (a *Attribute) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Numericoid: %s\n", string(a.numericoid))
	sb.WriteString("Names:")
	for n := range a.names {
		sb.WriteString(" " + n)
	}
	fmt.Fprintf(&sb, "\nDesc: %s\n", a.desc)
	fmt.Fprintf(&sb, "Obsolete: %t\n", a.obsolete)
	sb.WriteString("Sup Oid: ")
	if a.sup != nil {
		sb.WriteString(string(a.sup.Oid()))
	}
	sb.WriteRune('\n')

	sb.WriteString("Eq Rule: ")
	if a.eqRule != nil {
		sb.WriteString(string(a.eqRule.Syntax()))
	}
	sb.WriteRune('\n')

	sb.WriteString("Ord Rule: ")
	if a.ordRule != nil {
		sb.WriteString(string(a.ordRule.Syntax()))
	}
	sb.WriteRune('\n')

	sb.WriteString("Substr Rule: ")
	if a.subStrRule != nil {
		sb.WriteString(string(a.subStrRule.Syntax()))
	}
	sb.WriteRune('\n')

	fmt.Fprintf(&sb, "Syntax: %s\n", a.syntax)
	fmt.Fprintf(&sb, "Syntax len: %d\n", a.syntaxLen)
	fmt.Fprintf(&sb, "Single val: %t\n", a.singleVal)
	fmt.Fprintf(&sb, "Collective: %t\n", a.collective)
	fmt.Fprintf(&sb, "NoUserMod: %t\n", a.noUserMod)
	fmt.Fprintf(&sb, "Usage: %s\n", a.usage)

	return sb.String()
}

func AttributesAreEqual(a1, a2 *Attribute) error {
	if a1 == nil && a2 == nil {
		return nil
	}

	if a1 == nil {
		return fmt.Errorf("first attribute is nil")
	}

	if a2 == nil {
		return fmt.Errorf("second attribute is nil")
	}

	switch {
	case a1.numericoid != a2.numericoid:
		return fmt.Errorf("numericoids do not match")
	case !util.CmpMapKeys(a1.names, a2.names):
		return fmt.Errorf("names dont match")
	case a1.desc != a2.desc:
		return fmt.Errorf("descs dont match")
	case a1.obsolete != a2.obsolete:
		return fmt.Errorf("obsoletes dont match")

	case a1.eqRule != a2.eqRule:
		return fmt.Errorf("eqRules dont match")
	case a1.ordRule != a2.ordRule:
		return fmt.Errorf("ordRules dont match")
	case a1.subStrRule != a2.subStrRule:
		return fmt.Errorf("subStrRules dont match")
	case a1.syntax != a2.syntax:
		return fmt.Errorf("syntaxes dont match")
	case a1.syntaxLen != a2.syntaxLen:
		return fmt.Errorf("syntaxe lens dont match")
	case a1.singleVal != a2.singleVal:
		return fmt.Errorf("single vals dont match")
	case a1.collective != a2.collective:
		return fmt.Errorf("collectives dont match")
	case a1.noUserMod != a2.noUserMod:
		return fmt.Errorf("noUserMods dont match")
	case a1.usage != a2.usage:
		return fmt.Errorf("usages dont match")
	}

	if a1.sup == nil && a2.sup == nil {
		return nil
	}

	if a1.sup == nil || a2.sup == nil {
		return fmt.Errorf("sups dont match")
	}

	if a1.sup.numericoid != a2.sup.numericoid { // TODO nil checking??
		return fmt.Errorf("sups dont match")
	}

	return nil
}
