package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNodeNotLeaf  = errors.New("Node is not a leaf node")
	ErrUnknownScope = errors.New("Unknown scope")
)

type ResultCode int

const (
	Success                ResultCode = iota
	ProtocolError                     = 2
	AuthMethodNotSupported            = 7
	NoSuchAttribute                   = 16
	UndefinedAttributeType            = 17
	InappropriateMatching             = 18
	ConstraintViolation               = 19
	InvalidAttributeSyntax            = 21
	NoSuchObject                      = 32
	InvalidDnSyntax                   = 34
	InvalidCredentials                = 49
	UnwillingToPerform                = 53
	ObjectClassViolation              = 65
	Other                             = 80
)

func (rc ResultCode) String() string {
	switch rc {
	case Success:
		return "Success"
	case ProtocolError:
		return "ProtocolError"
	case AuthMethodNotSupported:
		return "AuthMethodNotSupported"
	case NoSuchAttribute:
		return "NoSuchAttribute"
	case UndefinedAttributeType:
		return "UndefinedAttributeType"
	case InappropriateMatching:
		return "InappropriateMatching"
	case ConstraintViolation:
		return "ConstraintViolation"
	case InvalidAttributeSyntax:
		return "InvalidAttributeSyntax"
	case NoSuchObject:
		return "NoSuchObject"
	case InvalidDnSyntax:
		return "InvalidDnSyntax"
	case InvalidCredentials:
		return "InvalidCredentials"
	case UnwillingToPerform:
		return "UnwillingToPerform"
	case ObjectClassViolation:
		return "ObjectClassViolation"
	case Other:
		return "Other"
	default:
		return fmt.Sprintf("unknown result code (%d)", rc)
	}
}

type LdapError struct {
	ResultCode        ResultCode
	MatchedDN         *DN
	DiagnosticMessage string
}

func NewLdapError(c ResultCode, matched *DN, format string, a ...any) LdapError {
	return LdapError{
		ResultCode:        c,
		MatchedDN:         matched,
		DiagnosticMessage: fmt.Sprintf(format, a...),
	}
}

func (e LdapError) Error() string {
	return fmt.Sprintf("LdapError code: %s (%d), matched: %q, msg: %s", e.ResultCode, e.ResultCode, e.MatchedDN, e.DiagnosticMessage)
}

// TODO should matching be more specific?
func (e LdapError) Is(target error) bool {
	lerr, ok := target.(LdapError)
	if !ok {
		return false
	}

	return e.ResultCode == lerr.ResultCode
}

type NodeNotFoundError struct {
	RequestedDN, MatchedDN DN
}

func (e NodeNotFoundError) Error() string {
	return fmt.Sprintf("requested DN: %s, matched up to: %s", e.RequestedDN, e.MatchedDN)
}

func (e *NodeNotFoundError) prependMatchedDn(rdn RDN) {
	e.MatchedDN.rdns = append([]RDN{rdn}, e.MatchedDN.rdns...)
}
