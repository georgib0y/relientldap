package schema

import (
	"fmt"
)

type MatchingRule interface {
	// Numericoid() string
	// Names() []string
	// Desc() string
	// Obsolete() bool
	Syntax() OID
	// match assumes that both attributes have the correct syntax
	Match(string, string) (bool, error)
}

// TODO do i need this info?
// type matchingRule struct {
// 	numericoid dit.OID
// 	names      []string
// 	desc       string
// 	obsolete   bool
// 	syntax     dit.OID
// 	// TODO extensions
// }

var UndefinedMatchingError error = fmt.Errorf("match is undefined")

var matchingRules map[OID]MatchingRule = map[OID]MatchingRule{
	"2.5.13.16": bitStringMatch{},
}

func GetMatchingRule(oid OID) (MatchingRule, bool) {
	r, ok := matchingRules[oid]
	return r, ok
}

type UnspecifiedMatchingRule struct{}

func (u UnspecifiedMatchingRule) Syntax() OID {
	return OID("")
}

func (u UnspecifiedMatchingRule) Match(s1, s2 string) (bool, error) {
	return s1 == s2, nil
}

type bitStringMatch struct{}

func (b bitStringMatch) Syntax() OID {
	return OID("1.3.6.1.4.1.1466.115.121.1.6")
}

func (b bitStringMatch) Match(s1, s2 string) (bool, error) {
	return s1 == s2, nil
}
