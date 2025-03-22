package schema

import (
	"fmt"

	"github.com/georgib0y/relientldap/internal/model/dit"
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
	numericoid                       dit.OID
	names                            map[string]bool
	desc                             string
	obsolete                         bool
	supOid                           dit.OID
	eqRule, ordRule, subStrRule      dit.OID
	syntax                           dit.OID
	syntaxLen                        int
	singleVal, collective, noUserMod bool
	usage                            UsageType
	extensions                       string
}

type AttrOption func(*Attribute)

func AttrWithOid(oid dit.OID) AttrOption {
	return func(a *Attribute) {
		a.numericoid = oid
	}
}

func AttrWithName(names ...string) AttrOption {
	return func(a *Attribute) {
		for _, n := range names {
			a.names[n] = true
		}
	}
}

func AttrWithDesc(desc string) AttrOption {
	return func(a *Attribute) {
		a.desc = desc
	}
}

func AttrWithObsolete() AttrOption {
	return func(a *Attribute) {
		a.obsolete = true
	}
}

func AttrWithSup(oid dit.OID) AttrOption {
	return func(a *Attribute) {
		a.supOid = oid
	}
}

func AttrWithEqRule(oid dit.OID) AttrOption {
	return func(a *Attribute) {
		a.eqRule = oid
	}
}

func AttrWithOrdRule(oid dit.OID) AttrOption {
	return func(a *Attribute) {
		a.ordRule = oid
	}
}

func AttrWithSubstrRule(oid dit.OID) AttrOption {
	return func(a *Attribute) {
		a.subStrRule = oid
	}
}

func AttrWithSyntax(oid dit.OID, len int) AttrOption {
	return func(a *Attribute) {
		a.syntax = oid
		a.syntaxLen = len
	}
}

func AttrWithSingleVal() AttrOption {
	return func(a *Attribute) {
		a.singleVal = true
	}
}

func AttrWithCollective() AttrOption {
	return func(a *Attribute) {
		a.collective = true
	}
}

func AttrWithNoUserMod() AttrOption {
	return func(a *Attribute) {
		a.noUserMod = true
	}
}

func AttrWithUsage(usage UsageType) AttrOption {
	return func(a *Attribute) {
		a.usage = usage
	}
}

func NewAttribute(opts ...AttrOption) Attribute {
	a := Attribute{
		names: map[string]bool{},
	}

	for _, opt := range opts {
		opt(&a)
	}

	return a
}
