package errors

type LDAPErrorType int

const (
	EntryAlreadyExists LDAPErrorType = iota
	EntryNotFound
	InvalidDNSyntax
	UndefinedAttributeType
)

type LDAPError struct {
	Code        LDAPErrorType
	dn, diagMsg string
}
