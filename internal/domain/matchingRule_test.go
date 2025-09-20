package domain

import (
	"testing"

	"github.com/georgib0y/relientldap/internal/util"
)

type matchingRuleTest struct {
	v1, v2 string
	exp    bool
	expErr error
}

func testMatchingRules(mrs []matchingRuleTest, mr MatchingRule, t *testing.T) {
	for _, m := range mrs {
		res, err := mr.Match(m.v1, m.v2)
		if err != m.expErr {
			t.Errorf("in matching %q and %q\texpected err: %s, got error: %s", m.v1, m.v2, m.expErr, err)
		}

		if res != m.exp {
			t.Errorf("in matching %q and %q\texpected res: %t, got: %t", m.v1, m.v2, m.exp, res)
		}
	}
}

func TestBitStringMatch(t *testing.T) {
	tests := []matchingRuleTest{
		{v1: "'00000'B", v2: "'00000'B", exp: true, expErr: nil},
		{v1: "'11111'B", v2: "'00000'B", exp: false, expErr: nil},
	}
	bitString := util.Unwrap(GetMatchingRule("bitStringMatch"))
	testMatchingRules(tests, bitString, t)
}

func TestCaseIgnoreMatch(t *testing.T) {
	tests := []matchingRuleTest{
		{v1: "abcd", v2: "abcd", exp: true, expErr: nil},
		{v1: "ABCD", v2: "abcd", exp: true, expErr: nil},
		{v1: "ABCdsgjklfds", v2: "abcd", exp: false, expErr: nil},
	}
	caseIgnore := util.Unwrap(GetMatchingRule("caseIgnoreMatch"))
	testMatchingRules(tests, caseIgnore, t)
}
