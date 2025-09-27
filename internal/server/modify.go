package server

import (
	"context"
	"io"
	"reflect"

	"github.com/georgib0y/relientldap/internal/app"
	d "github.com/georgib0y/relientldap/internal/domain"
	"github.com/georgib0y/relientldap/pkg/ber"
)

type Change struct {
	Operation    app.ModifyOperation `ber:"class=universal,cons=primitive,val=10"` //enumerated
	Modification PartialAttribute
}

func (c Change) ModOp() app.ModifyOperation {
	return c.Operation
}

func (c Change) Attribute() string {
	return c.Modification.AType
}

func (c Change) Vals() []string {
	vals := []string{}
	for v := range c.Modification.Vals {
		vals = append(vals, v)
	}
	return vals
}

type ModifyRequest struct {
	Object  string
	Changes []Change
}

func (m ModifyRequest) Dn() string {
	return m.Object
}

func (m ModifyRequest) Modifications() []app.Modification {
	mods := []app.Modification{}
	for _, c := range m.Changes {
		mods = append(mods, c)
	}
	return mods
}

func NewModifyResponse(msgId int, rc d.ResultCode, matchedDn, format string, a ...any) LdapMsg {
	return NewResultMsg(ModifyResponseTag, msgId, rc, matchedDn, format, a...)
}

type ModifyHandler struct {
	ms *app.ModifyService
}

func NewModifyHandler(ms *app.ModifyService) *ModifyHandler {
	return &ModifyHandler{ms}
}

func (m *ModifyHandler) RequestTag() ber.Tag {
	return ModifyRequestTag
}

func (m *ModifyHandler) ResponseTag() ber.Tag {
	return ModifyResponseTag
}

func (m *ModifyHandler) Handle(ctx context.Context, w io.Writer, msg LdapMsg) (err error) {
	var res LdapMsg
	defer func() {
		if err == nil {
			err = writeResponse(w, res)
		}
	}()

	logger.Print("in modify request")

	_, req, ok := msg.Request.Chosen()
	if !ok {
		res = NewModifyResponse(msg.MessageId, d.ProtocolError, "", "could not get modify req choice")
		return
	}

	mr, ok := req.(*ModifyRequest)
	if !ok {
		res = NewModifyResponse(
			msg.MessageId,
			d.ProtocolError,
			"",
			"expected *ModifyRequest, got %s", reflect.TypeOf(req),
		)
		return
	}

	modErr := m.ms.ModifyEntry(mr)
	if modErr != nil {
		err = modErr
		return
	}

	logger.Printf("modified entry: %s", mr.Dn())
	res = NewModifyResponse(msg.MessageId, d.Success, "", "modified entry at: %s", mr.Dn())
	return
}

type ModifyDnRequest struct {
	Entry        string
	NewRdn       string
	DeleteOldRdn bool
	NewSuperior  *ber.Optional[string]
}

func (mr ModifyDnRequest) Dn() string {
	return mr.Entry
}

func (mr ModifyDnRequest) UpdatedRdn() string {
	return mr.NewRdn
}

func (mr ModifyDnRequest) RemoveExistingRdn() bool {
	return mr.DeleteOldRdn
}

func (mr ModifyDnRequest) NewParentDn() (string, bool) {
	return mr.NewSuperior.Get()
}

func NewModifyDnResponse(msgId int, rc d.ResultCode, matchedDn, format string, a ...any) LdapMsg {
	return NewResultMsg(ModifyDnResponseTag, msgId, rc, matchedDn, format, a...)
}

type ModifyDnHandler struct {
	ms *app.ModifyService
}

func NewModifyDnHandler(ms *app.ModifyService) *ModifyDnHandler {
	return &ModifyDnHandler{ms}
}

func (m *ModifyDnHandler) RequestTag() ber.Tag {
	return ModifyDnRequestTag
}

func (m *ModifyDnHandler) ResponseTag() ber.Tag {
	return ModifyDnResponseTag
}

func (m *ModifyDnHandler) Handle(ctx context.Context, w io.Writer, msg LdapMsg) (err error) {
	var res LdapMsg
	defer func() {
		if err == nil {
			err = writeResponse(w, res)
		}
	}()

	logger.Print("in modify dn request")

	_, req, ok := msg.Request.Chosen()
	if !ok {
		res = NewModifyDnResponse(msg.MessageId, d.ProtocolError, "", "could not get modify dn req choice")
		return
	}

	mr, ok := req.(*ModifyDnRequest)
	if !ok {
		res = NewModifyDnResponse(
			msg.MessageId,
			d.ProtocolError,
			"",
			"expected *ModifyDnRequest, go %s", reflect.TypeOf(req),
		)
		return
	}

	modDnErr := m.ms.ModifyEntryDn(mr)
	if modDnErr != nil {
		err = modDnErr
		return
	}

	logger.Printf("modified dn entry: %s", mr.Dn())
	res = NewModifyDnResponse(msg.MessageId, d.Success, "", "modified entry at %s", mr.Dn())
	return
}
