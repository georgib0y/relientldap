package model

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
	NoSuchObject                      = 32
	InvalidDnSyntax                   = 34
	InvalidCredentials                = 49
	UnwillingToPerform                = 53
	ObjectClassViolation              = 65
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
	default:
		return "unknown result code"
	}
}

type LdapError struct {
	ResultCode        ResultCode
	MatchedDN         string
	DiagnosticMessage string
}

func NewLdapError(c ResultCode, matched string, format string, a ...any) LdapError {
	return LdapError{
		ResultCode:        c,
		MatchedDN:         matched,
		DiagnosticMessage: fmt.Sprintf(format, a...),
	}
}

func (e LdapError) Error() string {
	return fmt.Sprintf("LdapError code: %s (%d), matched: %s, msg: %s", e.ResultCode, e.ResultCode, e.MatchedDN, e.DiagnosticMessage)
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
