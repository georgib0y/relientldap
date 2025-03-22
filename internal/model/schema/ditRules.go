package schema

import "github.com/georgib0y/relientldap/internal/model/dit"

type DitContentRule struct {
	numericoid          dit.OID
	names               []string
	desc                string
	obsolete            bool
	aux, must, may, not []dit.OID
	//TODO extensions
}

type RuleId int

type DitStructureRule struct {
	numericoid dit.OID
	names      []string
	desc       string
	obsolete   bool
	form       dit.OID
	sup        []RuleId
	// TODO extensions
}

type NameForm struct {
	numericoid dit.OID
	names      []string
	desc       string
	obsolete   bool
	oc         dit.OID
	must, may  []dit.OID
	// TODO extensions
}
