package server

import (
	"fmt"

	d "github.com/georgib0y/relientldap/internal/domain"
	"github.com/georgib0y/relientldap/pkg/ber"
)

type PartialAttribute struct {
	AType string
	Vals  ber.Set[string]
}

var (
	BindRequestTag  = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 0}
	BindResponseTag = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 1}
	BrSimpleTag     = ber.Tag{Class: ber.ContextSpecific, Construct: ber.Constructed, Value: 0}
	BrSaslTag       = ber.Tag{Class: ber.ContextSpecific, Construct: ber.Constructed, Value: 3}

	UnbindRequestTag = ber.Tag{Class: ber.Application, Construct: ber.Primitive, Value: 2}

	ModifyRequestTag  = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 6}
	ModifyResponseTag = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 7}

	AddRequestTag  = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 8}
	AddResponseTag = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 9}

	ModifyDnRequestTag  = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 12}
	ModifyDnResponseTag = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 13}
)

type LdapMsgChoice struct {
	BindRequest BindRequest `ber:"class=application,cons=constructed,val=0"`
	// TODO proper BindResponse type
	BindResponse  LdapResult `ber:"class=application,cons=constructed,val=1"`
	UnbindRequest string     `ber:"class=application,cons=primitive,val=2"`

	ModifyRequest  ModifyRequest `ber:"class=application,cons=constructed,val=6"`
	ModifyResponse LdapResult    `ber:"class=application,cons=constructed,val=7"`

	AddRequest  AddRequest `ber:"class=application,cons=constructed,val=8"`
	AddResponse LdapResult `ber:"class=application,cons=constructed,val=9"`

	ModifyDnRequest  ModifyDnRequest `ber:"class=application,cons=constructed,val=12"`
	ModifyDnResponse LdapResult      `ber:"class=application,cons=constructed,val=13"`
}

type LdapMsg struct {
	MessageId int
	Request   *ber.Choice[LdapMsgChoice]
	Controls  *ber.Optional[[]byte] `ber:"class=context-specific,cons=constructed,val=0"`
}

// TODO implement embedded structs for en/decoding so i dont have to continually repeat myself
type LdapResult struct {
	ResultCode        d.ResultCode `ber:"class=universal,cons=primitive,val=10"` // enumerated
	MatchedDN         string
	DiagnosticMessage string
	Referral          *ber.Optional[[]byte]
}

func NewResultMsg(tag ber.Tag, msgId int, rc d.ResultCode, matchedDn, format string, a ...any) LdapMsg {
	res := LdapResult{
		ResultCode:        rc,
		MatchedDN:         matchedDn,
		DiagnosticMessage: fmt.Sprintf(format, a...),
	}

	return LdapMsg{
		MessageId: msgId,
		Request:   ber.NewChosen[LdapMsgChoice](tag, res),
	}
}
