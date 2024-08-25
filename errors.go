package main

import "fmt"

type LDAPError interface {
	Result() Result
}

type EntryAleadyExistsError struct {
	dn  DN
	msg string
}

func (e EntryAleadyExistsError) Error() string {
	return fmt.Sprintf("Entry already exists at: %s. %s", e.dn, e.msg)
}

func (e EntryAleadyExistsError) Result() Result {
	return Result{
		ResultCode:        EntryAlreadyExists,
		MatchedDN:         e.dn.String(),
		DiagnosticMessage: e.msg,
	}
}

type EntryNotFoundError struct {
	dn  DN
	msg string
}

func (e EntryNotFoundError) Error() string {
	return fmt.Sprintf("Could not find entity: %s. %s", e.dn, e.msg)
}

func (e EntryNotFoundError) Result() Result {
	return Result{
		ResultCode:        NoSuchObject,
		MatchedDN:         e.dn.String(),
		DiagnosticMessage: e.msg,
	}
}

type InvalidDNSyntaxError struct {
	dn  string
	msg string
}

func (i InvalidDNSyntaxError) Error() string {
	return fmt.Sprintf("Invalid DN syntax: %s. %s", i.dn, i.msg)
}

func (i InvalidDNSyntaxError) Result() Result {
	return Result{
		ResultCode:        InvalidDNSyntax,
		MatchedDN:         i.dn,
		DiagnosticMessage: i.msg,
	}
}

type UndefinedAttributeTypeError struct {
	attr, dn, msg string
}

func (u UndefinedAttributeTypeError) Error() string {
	return fmt.Sprintf("Attr %s not found in schema. %s", u.attr, u.msg)
}

func (u UndefinedAttributeTypeError) Result() Result {
	return Result{
		ResultCode:        UndefinedAttributeType,
		MatchedDN:         u.dn,
		DiagnosticMessage: u.msg,
	}
}
