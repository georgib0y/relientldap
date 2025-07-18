package server

import (
	"github.com/georgib0y/relientldap/pkg/ber"
)

var (
	BindRequestTag  = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 0}
	BindResponseTag = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 1}
	BrSimpleTag     = ber.Tag{Class: ber.ContextSpecific, Construct: ber.Constructed, Value: 0}
	BrSaslTag       = ber.Tag{Class: ber.ContextSpecific, Construct: ber.Constructed, Value: 3}
	BindRespTag     = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 1}

	UnbindRequestTag = ber.Tag{Class: ber.Application, Construct: ber.Primitive, Value: 2}
)

type LdapMsg struct {
	MessageId int
	Request   *ber.Choice[LdapMsgChoice]
	Controls  *ber.Optional[[]byte] `ber:"class=context-specific,cons=constructed,val=0"`
}

type LdapMsgChoice struct {
	BindRequest   BindRequest  `ber:"class=application,cons=constructed,val=0"`
	BindResponse  BindResponse `ber:"class=application,cons=constructed,val=1"`
	UnbindRequest string       `ber:"class=application,cons=primitive,val=2"`
}

type ResultCode int

const (
	Success                ResultCode = iota
	ProtocolError                     = 2
	AuthMethodNotSupported            = 7
	NoSuchObject                      = 32
)

// TODO implement embedded structs for en/decoding so i dont have to continually repeat myself
// type LdapResult struct {
// 	ResultCode        ResultCode `ber:"class=universal,cons=primitive,val=10"` // enumerated
// 	MatchedDN         string
// 	DiagnosticMessage string
// 	Referral          *ber.Optional[[]byte]
// }

type BindRequest struct {
	Version int
	Name    string
	Auth    *ber.Choice[BindReqChoice]
}

type BindReqChoice struct {
	Simple string   `ber:"class=context-specific,cons=constructed,val=0"`
	Sasl   SaslAuth `ber:"class=context-specific,cons=constructed,val=3"`
}

type SaslAuth struct {
	Mechanism   string
	Credentials string
}

type BindResponse struct {
	ResultCode        ResultCode `ber:"class=universal,cons=primitive,val=10"` // enumerated
	MatchedDN         string
	DiagnosticMessage string
	Referral          *ber.Optional[[]byte]
	ServerSaslCreds   *ber.Optional[string]
}

// TODO server sasl creds
func NewBindResponse(msgId int, rc ResultCode, matchedDn, diagnostic string) LdapMsg {
	resp := BindResponse{
		ResultCode:        rc,
		MatchedDN:         matchedDn,
		DiagnosticMessage: diagnostic,
	}

	return LdapMsg{
		MessageId: msgId,
		Request:   ber.NewChosen[LdapMsgChoice](BindRespTag, resp),
	}
}
