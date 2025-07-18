package model

import (
	"fmt"
	"log"
	"strings"
)

var (
	UndefinedMatch            = fmt.Errorf("match is undefined")
	UnimplementedMatchingRule = fmt.Errorf("unimplemented matching rule")
)

// TODO names might actually need to be an array of strings as per the rfc
type MatchingRule struct {
	numericoid OID
	name       string
	syntax     OID
	match      func(string, string) (bool, error)
}

func (m MatchingRule) Oid() OID {
	return m.numericoid
}

func (m MatchingRule) Name() string {
	return m.name
}

func (m MatchingRule) Syntax() OID {
	return m.syntax
}

func (m MatchingRule) Match(v1, v2 string) (bool, error) {
	if m.match == nil {
		logger.Panic("tried to call match on a zeroed matchingrule")
	}
	return m.match(v1, v2)
}

var matchingRules = map[string]MatchingRule{
	"bitStringMatch": MatchingRule{
		numericoid: "2.5.13.16",
		name:       "bitStringMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.6",
		match:      bitStringMatch,
	},
	"caseIgnoreIA5Match": MatchingRule{
		numericoid: "1.3.6.1.4.1.1466.109.114.2",
		name:       "caseIgnoreIA5Match",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.26",
		match:      caseIgnoreMatch,
	},
	"caseIgnoreIA5SubstringsMatch": MatchingRule{
		numericoid: "1.3.6.1.4.1.1466.109.114.3",
		name:       "caseIgnoreIA5SubstringsMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.58",
		match:      unimplementedMatch,
	},
	"caseIgnoreListMatch": MatchingRule{
		numericoid: "2.5.13.11",
		name:       "caseIgnoreListMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.41",
		match:      unimplementedMatch,
	},
	"caseIgnoreListSubstringsMatch": MatchingRule{
		numericoid: "2.5.13.12",
		name:       "caseIgnoreListSubstringsMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.58",
		match:      unimplementedMatch,
	},
	"caseIgnoreMatch": MatchingRule{
		numericoid: "2.5.13.2",
		name:       "caseIgnoreMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.15",
		match:      caseIgnoreMatch,
	},
	"caseIgnoreSubstringsMatch": MatchingRule{
		numericoid: "2.5.13.4",
		name:       "caseIgnoreSubstringsMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.58",
		match:      unimplementedMatch,
	},
	"caseIgnoreOrderingMatch": MatchingRule{
		numericoid: "2.5.13.3",
		name:       "caseIgnoreOrderingMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.15",
		match:      unimplementedMatch,
	},
	"distinguishedNameMatch": MatchingRule{
		numericoid: "2.5.13.1",
		name:       "distinguishedNameMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.12",
		match:      unimplementedMatch,
	},
	"numericStringMatch": MatchingRule{
		numericoid: "2.5.13.8",
		name:       "numericStringMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.36",
		match:      unimplementedMatch,
	},
	"numericStringSubstringsMatch": MatchingRule{
		numericoid: "2.5.13.10",
		name:       "numericStringSubstringsMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.58",
		match:      unimplementedMatch,
	},
	"octetStringMatch": MatchingRule{
		numericoid: "2.5.13.17",
		name:       "octetStringMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.40",
		match:      unimplementedMatch,
	},
	"telephoneNumberMatch": MatchingRule{
		numericoid: "2.5.13.20",
		name:       "telephoneNumberMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.50",
		match:      unimplementedMatch,
	},
	"telephoneNumberSubstringsMatch": MatchingRule{
		numericoid: "2.5.13.21",
		name:       "telephoneNumberSubstringsMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.58",
		match:      unimplementedMatch,
	},
	"uniqueMemberMatch": MatchingRule{
		numericoid: "2.5.13.23",
		name:       "uniqueMemberMatch",
		syntax:     "1.3.6.1.4.1.1466.115.121.1.34",
		match:      unimplementedMatch,
	},
}

func GetMatchingRule(nameOrOid string) (*MatchingRule, bool) {
	if mr, ok := matchingRules[nameOrOid]; ok {
		return &mr, true
	}

	for _, mr := range matchingRules {
		if mr.numericoid == OID(nameOrOid) {
			return &mr, true
		}
	}

	return nil, false
}

// For usage in tests where the name or oid is known, panics if the
// name/oid is not known to be good, use GetMatchingRule elsewhere
func GetMatchingRuleUnchecked(nameOrOid string) *MatchingRule {
	mr, ok := GetMatchingRule(nameOrOid)
	if !ok {
		log.Panicf("unknown matching rule name/oid: %s", nameOrOid)
	}
	return mr
}

func unimplementedMatch(s1, s2 string) (bool, error) {
	logger.Print("matching rule is unimplemented")
	return false, UnimplementedMatchingRule
}

func bitStringMatch(s1, s2 string) (bool, error) {
	return s1 == s2, nil
}

// TODO insignificant space handling
func caseIgnoreMatch(s1, s2 string) (bool, error) {
	return strings.ToLower(s1) == strings.ToLower(s2), nil
}

type substringAssertion struct {
	initial string
	any     []string
	final   string
}

// TODO asterisk and backslash escaping
// func parseSubstringAssertion(s string) (substringAssertion, error) {

// }

// func matchSubstr(subAssert, val string) (bool, error) {

// }
