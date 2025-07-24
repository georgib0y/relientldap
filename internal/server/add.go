package server

import (
	"context"
	"io"
	"reflect"

	"github.com/georgib0y/relientldap/internal/app"
	d "github.com/georgib0y/relientldap/internal/domain"
	"github.com/georgib0y/relientldap/pkg/ber"
)

// the real attribute deinition requies that vals is not empty
type Attribute struct {
	AType string
	Vals  ber.Set[string]
}

type AddRequest struct {
	Entry string
	Attrs []Attribute
}

func (ar AddRequest) Dn() string {
	return ar.Entry
}

func (ar AddRequest) Attributes() map[string][]string {
	attrs := map[string][]string{}
	for _, a := range ar.Attrs {
		vals := []string{}
		for v := range a.Vals {
			vals = append(vals, v)
		}

		attrs[a.AType] = vals
	}

	return attrs
}

func NewAddResponse(msgId int, rc d.ResultCode, matchedDn, format string, a ...any) LdapMsg {
	return NewResultMsg(AddResponseTag, msgId, rc, matchedDn, format, a...)
}

type AddHandler struct {
	as *app.AddService
}

func NewAddHandler(as *app.AddService) *AddHandler {
	return &AddHandler{as}
}

func (a *AddHandler) RequestTag() ber.Tag {
	return AddRequestTag
}

func (a *AddHandler) ResponseTag() ber.Tag {
	return AddResponseTag
}

func (a *AddHandler) Handle(ctx context.Context, w io.Writer, msg LdapMsg) (err error) {
	var res LdapMsg
	defer func() {
		if err == nil {
			err = writeResponse(w, res)
		}
	}()

	logger.Print("in add request")

	_, req, ok := msg.Request.Chosen()
	if !ok {
		res = NewAddResponse(msg.MessageId, d.ProtocolError, "", "could not get add request choice")
		return
	}

	ar, ok := req.(*AddRequest)
	if !ok {
		res = NewAddResponse(
			msg.MessageId,
			d.ProtocolError,
			"",
			"expected *AddRequest, got %s", reflect.TypeOf(req),
		)
		return
	}

	entry, addErr := a.as.AddEntry(ar)
	if addErr != nil {
		err = addErr
		return
	}

	logger.Printf("added entry: %s", entry)
	res = NewAddResponse(msg.MessageId, d.Success, "", "created entry at: %s", ar.Dn())
	return
}
