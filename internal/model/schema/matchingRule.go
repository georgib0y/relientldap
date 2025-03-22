package schema

import "github.com/georgib0y/relientldap/internal/model/dit"

type MatchingRule struct {
	numericoid dit.OID
	names      []string
	desc       string
	obsolete   bool
	syntax     dit.OID
	// TODO extensions
}

type MatchingRuleUse struct {
	numericoid dit.OID
	names      []string
	desc       string
	obsolete   bool
	applies    []dit.OID
	// TODO extensions
}

type LdapSyntax struct {
	numericoid dit.OID
	desc       string
	// TODO extensions
}
