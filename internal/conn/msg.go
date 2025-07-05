package conn

import (
	"fmt"

	"github.com/georgib0y/relientldap/pkg/ber"
)

type LdapMsg struct {
	MessageId int
	Request   *LdapMsgChoice
}

var (
	BindRequestTag = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 0}
)

type LdapMsgChoice struct {
	t           *ber.Tag
	BindRequest BindRequest
}

func NewLdapBrMsgChoice(br BindRequest) *LdapMsgChoice {
	var l LdapMsgChoice
	l.t = new(ber.Tag)
	*l.t = BindRequestTag
	l.BindRequest = br
	return &l
}

func (l *LdapMsgChoice) Choose(t ber.Tag) (any, error) {
	var choice any
	switch t {
	case BindRequestTag:
		choice = &l.BindRequest
	default:
		return nil, fmt.Errorf("unknown tag: %q for ldap msg", t)
	}

}

func (l *LdapMsgChoice) Tag() (ber.Tag, bool) {
	if l.t == nil {
		return ber.Tag{}, false
	}

	return *l.t, true
}
