package domain

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/georgib0y/relientldap/internal/util"
)

var (
	ObjectClassAttribute = NewAttributeBuilder().
		SetOid("2.5.4.0").
		AddNames("objectClass").
		SetEqRule(util.Unwrap(GetMatchingRule("objectIdentifierMatch"))).
		SetSyntax(util.Unwrap(GetSyntax("1.3.6.1.4.1.1466.115.121.1.38")), 0).
		Build()
	// TODO
	// creatorsName
	// createTimestamp
	// modifiersName
	// modifyTimestamp
	// struturalObjectClass
	// governingStructureRule

	// altServer
	// namingContexts
	// supportedControl
	// supportedExtensions
	// supportedFeatures
	// supportedLDAPVersion
	// supportedSASLMechanism
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
	friendlyName                     string
	desc                             string
	obsolete                         bool
	sup                              *Attribute
	eqRule, ordRule, subStrRule      MatchingRule
	syntax                           Syntax
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
		if b.a.friendlyName == "" || len(b.a.friendlyName) > len(n) {
			b.a.friendlyName = n
		}
		b.a.names[n] = struct{}{}
	}
	return b
}

func (b *AttributeBuilder) SetDesc(desc string) *AttributeBuilder {
	b.a.desc = desc
	return b
}

func (b *AttributeBuilder) SetObsolete(o bool) *AttributeBuilder {
	b.a.obsolete = o
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

func (b *AttributeBuilder) SetSyntax(syntax Syntax, len int) *AttributeBuilder {
	b.a.syntax = syntax
	b.a.syntaxLen = len
	return b
}

func (b *AttributeBuilder) SetSyntaxLength(len int) *AttributeBuilder {
	b.a.syntaxLen = len
	return b
}

func (b *AttributeBuilder) SetSingleVal(s bool) *AttributeBuilder {
	b.a.singleVal = s
	return b
}

func (b *AttributeBuilder) SetCollective(c bool) *AttributeBuilder {
	b.a.collective = c
	return b
}

func (b *AttributeBuilder) SetNoUserMod(n bool) *AttributeBuilder {
	b.a.noUserMod = n
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

func (a *Attribute) Name() string {
	if a.friendlyName != "" {
		return a.friendlyName
	}
	return string(a.numericoid)
}

func (a *Attribute) HasName(name string) bool {
	_, ok := a.names[name]
	return ok
}

func (a *Attribute) Syntax() (Syntax, int, bool) {
	var zero Syntax
	for a != nil {
		if !a.syntax.Eq(zero) {
			return a.syntax, a.syntaxLen, true
		}
		a = a.sup
	}

	return zero, 0, false
}

func (a *Attribute) EqRule() (MatchingRule, bool) {
	var zero MatchingRule
	// if the current attribute does not have an eq rule, the sup(s) might
	for a != nil {
		if !a.eqRule.Eq(zero) {
			return a.eqRule, true
		}
		a = a.sup
	}
	return zero, false
}

func (a *Attribute) SingleVal() bool {
	return a.singleVal
}

func (a *Attribute) String() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Numericoid: %q\n", string(a.numericoid))
	sb.WriteString("Names:")
	for n := range a.names {
		fmt.Fprintf(&sb, " %q", n)
	}
	fmt.Fprintf(&sb, "\nDesc: %q\n", a.desc)
	fmt.Fprintf(&sb, "Obsolete: %t\n", a.obsolete)
	sb.WriteString("Sup Oid: ")
	if a.sup != nil {
		fmt.Fprintf(&sb, " %q", string(a.sup.Oid()))
	}
	sb.WriteRune('\n')

	var zero MatchingRule
	sb.WriteString("Eq Rule: ")
	if !a.eqRule.Eq(zero) {
		fmt.Fprintf(&sb, " %q", string(a.eqRule.Syntax()))
	}
	sb.WriteRune('\n')

	sb.WriteString("Ord Rule: ")
	if !a.ordRule.Eq(zero) {
		fmt.Fprintf(&sb, " %q", string(a.ordRule.Syntax()))
	}
	sb.WriteRune('\n')

	sb.WriteString("Substr Rule: ")
	if !a.subStrRule.Eq(zero) {
		fmt.Fprintf(&sb, " %q", string(a.subStrRule.Syntax()))
	}
	sb.WriteRune('\n')

	fmt.Fprintf(&sb, "Syntax: %q\n", a.syntax.numericoid)
	fmt.Fprintf(&sb, "Syntax len: %d\n", a.syntaxLen)
	fmt.Fprintf(&sb, "Single val: %t\n", a.singleVal)
	fmt.Fprintf(&sb, "Collective: %t\n", a.collective)
	fmt.Fprintf(&sb, "NoUserMod: %t\n", a.noUserMod)
	fmt.Fprintf(&sb, "Usage: %q\n", a.usage)

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

	case a1.eqRule.numericoid != a2.eqRule.numericoid:
		return fmt.Errorf("eqRules dont match, %s\n%s", a1.eqRule, a2.eqRule)
	case a1.ordRule.numericoid != a2.ordRule.numericoid:
		return fmt.Errorf("ordRules dont match")
	case a1.subStrRule.numericoid != a2.subStrRule.numericoid:
		return fmt.Errorf("subStrRules dont match")
	case !a1.syntax.Eq(a2.syntax):
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

func ParseAttributes(r io.Reader) (map[OID]*Attribute, error) {
	tokeniser, err := NewTokeniser(r)
	if err != nil {
		return nil, err
	}

	ldifs := []*LdifAttribute{}
	for tokeniser.HasNext() {
		subt, err := tokeniser.ParenSubTokeniser()
		if err != nil {
			return nil, err
		}
		l, err := NewLdifAttribute(subt)
		if err != nil {
			return nil, err
		}
		ldifs = append(ldifs, l)
	}

	resolver := NewLdifAttrResolver(ldifs)
	return resolver.Resolve()
}

type LdifAttrResolver struct {
	ldif  []*LdifAttribute
	attrs map[OID]*Attribute
}

func NewLdifAttrResolver(ldifs []*LdifAttribute) *LdifAttrResolver {
	return &LdifAttrResolver{ldifs, map[OID]*Attribute{}}
}

func (r *LdifAttrResolver) Resolve() (map[OID]*Attribute, error) {
	for _, l := range r.ldif {
		if err := l.Build(r); err != nil {
			return nil, err
		}
	}

	return r.attrs, nil
}

func (r *LdifAttrResolver) supByOid(o OID) *Attribute {
	sup, ok := r.attrs[o]
	if ok {
		return sup
	}

	sup = new(Attribute)
	r.attrs[o] = sup
	return sup
}

func (r *LdifAttrResolver) GetSup(nameOrOid string) (*Attribute, error) {
	for _, a := range r.ldif {
		if a.numericoid == nameOrOid {
			return r.supByOid(OID(a.numericoid)), nil
		}
		for _, n := range a.names {
			if nameOrOid == n {
				return r.supByOid(OID(a.numericoid)), nil
			}
		}
	}

	return nil, fmt.Errorf("could not find attr sup %q", nameOrOid)
}

func (r *LdifAttrResolver) PutAttr(attr *Attribute) {
	if a, ok := r.attrs[attr.Oid()]; ok {
		*a = *attr
	} else {
		r.attrs[attr.Oid()] = attr
	}
}

type LdifAttribute struct {
	numericoid      string
	names           []string
	desc            string
	obsolete        bool
	sup             string
	eq, ord, substr string
	syntax          string
	syntaxLen       int
	singleVal       bool
	collective      bool
	noUserMod       bool
	usage           string
	// TODO extensions
}

func NewLdifAttribute(t *Tokeniser) (*LdifAttribute, error) {
	attr := &LdifAttribute{
		names: []string{},
	}

	if err := attr.setNumericoid(t); err != nil {
		return nil, err
	}

	keywords := []string{
		"NAME",
		"DESC",
		"OBSOLETE",
		"SUP",
		"EQUALITY",
		"ORDERING",
		"SUBSTR",
		"SYNTAX",
		"SINGLE-VALUE",
		"COLLECTIVE",
		"NO-USER-MODIFICATION",
		"USAGE",
	}

	for len(keywords) > 0 {
		keyword, ok := t.Next()
		if !ok {
			return attr, nil
		}

		if keyword.tokenType != KEYWORD {
			return nil, fmt.Errorf("expected KEYWORD got %s (%s)", keyword.tokenType, keyword.val)
		}

		idx := slices.Index(keywords, keyword.val)
		if idx == -1 {
			return nil, fmt.Errorf("unknown keyword or in unexpected position %q", keyword.val)
		}
		keywords = keywords[idx+1:]

		var err error
		switch keyword.val {
		case "NAME":
			err = attr.setName(t)
		case "DESC":
			err = attr.setDesc(t)
		case "OBSOLETE":
			err = attr.setObsolete()
		case "SUP":
			err = attr.setSup(t)
		case "EQUALITY":
			err = attr.setEq(t)
		case "ORDERING":
			err = attr.setOrdering(t)
		case "SUBSTR":
			err = attr.setSubstr(t)
		case "SYNTAX":
			err = attr.setSyntax(t)
		case "SINGLE-VALUE":
			err = attr.setSingleVal()
		case "COLLECTIVE":
			err = attr.setCollective()
		case "NO-USER-MODIFICATION":
			err = attr.setNoUserMod()
		case "USAGE":
			err = attr.setUsage(t)
		default:
			return nil, fmt.Errorf("unknown attribute keyword %q", keyword.val)
		}

		if err != nil {
			return nil, err
		}
	}

	return attr, nil
}

// builds and places this attribute into the attribute map, uses the others slice to get references to sups
func (a *LdifAttribute) Build(r *LdifAttrResolver) error {
	b := NewAttributeBuilder()
	b.SetOid(OID(a.numericoid)).
		AddNames(a.names...).
		SetDesc(a.desc).
		SetObsolete(a.obsolete).
		SetSingleVal(a.singleVal).
		SetCollective(a.collective).
		SetNoUserMod(a.noUserMod)

	if a.syntax != "" {
		s, err := GetSyntax(OID(a.syntax))
		if err != nil {
			return err
		}
		b.SetSyntax(s, a.syntaxLen)
	}

	if a.sup != "" {
		sup, err := r.GetSup(a.sup)
		if err != nil {
			return err
		}
		b.SetSup(sup)
	}

	if a.eq != "" {
		eq, err := GetMatchingRule(a.eq)
		if err != nil {
			return err
		}
		b.SetEqRule(eq)
	}

	if a.ord != "" {
		ord, err := GetMatchingRule(a.ord)
		if err != nil {
			return err
		}
		b.SetOrdRule(ord)
	}

	if a.substr != "" {
		sub, err := GetMatchingRule(a.substr)
		if err != nil {
			return err
		}
		b.SetSubStrRule(sub)
	}

	if a.usage != "" {
		usage, err := NewUsage(a.usage)
		if err != nil {
			return err
		}
		b.SetUsage(usage)
	}

	r.PutAttr(b.Build())
	return nil
}

func (a *LdifAttribute) setNumericoid(t *Tokeniser) error {
	numericoid, err := t.NextNumericoid()
	if err != nil {
		return err
	}
	a.numericoid = numericoid.val
	return nil
}

func (a *LdifAttribute) setName(t *Tokeniser) error {
	tokens, err := t.NextQdescrs()
	if err != nil {
		return err
	}

	for _, token := range tokens {
		a.names = append(a.names, stripQuotes(token.val))
	}
	return nil
}

func (a *LdifAttribute) setDesc(t *Tokeniser) error {
	desc, err := t.NextQdstring()
	if err != nil {
		return err
	}
	a.desc = stripQuotes(desc.val)
	return nil
}

func (a *LdifAttribute) setObsolete() error {
	a.obsolete = true
	return nil
}

func (a *LdifAttribute) setSup(t *Tokeniser) error {
	sup, err := t.NextOid()
	if err != nil {
		return err
	}
	a.sup = stripQuotes(sup.val)
	return nil
}

func (a *LdifAttribute) setEq(t *Tokeniser) error {
	eq, err := t.NextOid()
	if err != nil {
		return err
	}
	a.eq = stripQuotes(eq.val)
	return nil
}

func (a *LdifAttribute) setOrdering(t *Tokeniser) error {
	ord, err := t.NextOid()
	if err != nil {
		return err
	}
	a.ord = stripQuotes(ord.val)
	return nil
}

func (a *LdifAttribute) setSubstr(t *Tokeniser) error {
	sub, err := t.NextOid()
	if err != nil {
		return err
	}
	a.substr = stripQuotes(sub.val)
	return nil
}

func (a *LdifAttribute) setSyntax(t *Tokeniser) error {
	stx, err := t.NextNoidlen()
	if err != nil {
		return err
	}
	// TODO handle if noidlen curly brackets
	if stx.tokenType == NUMERICOID {
		a.syntax = stripQuotes(stx.val)
		return nil
	}

	spl := strings.Split(stx.val, "{")
	oid := spl[0]
	len, err := strconv.Atoi(spl[1][:len(spl[1])-1])
	if err != nil {
		return err
	}

	a.syntax = oid
	a.syntaxLen = len
	return nil
}

func (a *LdifAttribute) setSingleVal() error {
	a.singleVal = true
	return nil
}

func (a *LdifAttribute) setCollective() error {
	a.collective = true
	return nil
}

func (a *LdifAttribute) setNoUserMod() error {
	a.noUserMod = true
	return nil
}

func (a *LdifAttribute) setUsage(t *Tokeniser) error {
	usage, err := t.NextDescr()
	if err != nil {
		return err
	}
	a.usage = stripQuotes(usage.val)
	return nil
}
