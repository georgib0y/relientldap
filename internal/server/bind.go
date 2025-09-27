package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"

	"github.com/georgib0y/relientldap/internal/app"
	d "github.com/georgib0y/relientldap/internal/domain"
	"github.com/georgib0y/relientldap/pkg/ber"
)

type BindRequest struct {
	Ver  int
	Name string
	Auth *ber.Choice[BindReqChoice]
}

type BindReqChoice struct {
	Simple string   `ber:"class=context-specific,cons=constructed,val=0"`
	Sasl   SaslAuth `ber:"class=context-specific,cons=constructed,val=3"`
}

type SaslAuth struct {
	Mechanism   string
	Credentials string
}

func (br BindRequest) Version() int {
	return br.Ver
}

func (br BindRequest) Dn() string {
	return br.Name
}

func (br BindRequest) Simple() (string, bool) {
	if _, auth, ok := br.Auth.Chosen(); ok {
		if simple, ok := auth.(*string); ok {
			return *simple, true
		}

	}

	return "", false
}

func (br BindRequest) SaslMechanism() (string, bool) {
	if _, auth, ok := br.Auth.Chosen(); ok {
		if sasl, ok := auth.(*SaslAuth); ok {
			return sasl.Mechanism, true
		}

	}

	return "", false
}

func (br BindRequest) SaslCredentials() (string, bool) {
	if _, auth, ok := br.Auth.Chosen(); ok {
		if sasl, ok := auth.(*SaslAuth); ok {
			return sasl.Credentials, true
		}

	}

	return "", false
}

type BindHandler struct {
	bs app.BindService
}

func NewBindHandler(bs app.BindService) *BindHandler {
	return &BindHandler{bs}
}

func (h *BindHandler) RequestTag() ber.Tag {
	return BindRequestTag
}

func (h *BindHandler) ResponseTag() ber.Tag {
	return BindResponseTag
}

func (b *BindHandler) Handle(ctx context.Context, w io.Writer, msg LdapMsg) (err error) {
	var res LdapMsg
	defer func() {
		if err == nil {
			err = writeResponse(w, res)
		}
	}()

	logger.Print("in bind request")

	_, req, ok := msg.Request.Chosen()
	if !ok {
		res = NewResultMsg(BindResponseTag,
			msg.MessageId,
			d.ProtocolError,
			"",
			"could not get choice for bind request",
		)
		return
	}

	br, ok := req.(*BindRequest)
	if !ok {
		res = NewResultMsg(BindResponseTag,
			msg.MessageId,
			d.ProtocolError,
			"",
			"expected %s, got %s", reflect.TypeFor[BindRequest](), reflect.TypeOf(req),
		)
		return
	}
	logger.Print("extracted bind request")

	entry, autherr := b.bs.Bind(br)
	_ = autherr
	if lerr, ok := autherr.(d.LdapError); ok {
		logger.Print("caught ldaperror in simple")
		res = NewResultMsg(BindResponseTag,
			msg.MessageId,
			lerr.ResultCode,
			lerr.MatchedDN.String(),
			"%s",
			lerr.DiagnosticMessage,
		)
		return
	} else if autherr != nil {
		logger.Print("caught regular error in simple")
		err = autherr
		return
	}

	logger.Print("auth success")

	// TODO this is probably horribly unthreadsafe
	boundEntryVal := ctx.Value(BoundEntryKey)
	be, ok := boundEntryVal.(**d.Entry)
	if !ok {
		return fmt.Errorf("bound entry did not exist or was not an entry pointer pointer")
	}
	*be = entry

	res = NewResultMsg(BindResponseTag,
		msg.MessageId,
		d.Success,
		br.Name,
		"bind request successfult (no password checking yet)",
	)

	return
}

var UnbindError = errors.New("unbind request recieved")

var UnbindHandler = HandleFunc(UnbindRequestTag, UnbindRequestTag, func(ctx context.Context, w io.Writer, msg LdapMsg) error {
	return UnbindError
})
