package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"reflect"

	d "github.com/georgib0y/relientldap/internal/domain"
	"github.com/georgib0y/relientldap/pkg/ber"
)

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

func getAuthenticatedContext(ctx context.Context, s *Scheduler, dn d.DN) (context.Context, error) {
	done := make(chan context.Context)
	errChan := make(chan error)

	s.Schedule(func(d d.DIT) {
		if _, err := d.GetEntry(dn); err != nil {
			errChan <- err
			return
		}

		newCtx := context.WithValue(ctx, BoundDnKey, dn)
		done <- newCtx
	})

	// wait for the scheduled fn to finish
	select {
	case newCtx := <-done:
		return newCtx, nil
	case err := <-errChan:
		return ctx, err
	}

}

// TODO refactor
func HandleBindRequest(ctx context.Context, w io.Writer, s *Scheduler, msg LdapMsg) (context.Context, error) {
	logger.Print("in bind request")

	// TODO check version and send protocolerror

	_, req, ok := msg.Request.Chosen()
	if !ok {
		return ctx, fmt.Errorf("could not get choice for bind request")
	}

	br, ok := req.(*BindRequest)
	if !ok {
		return ctx, fmt.Errorf("expected *BindRequest, got %s", reflect.TypeOf(req))
	}
	logger.Print("extracted bind request")

	// dn, err := d.NormaliseDN(s.s, br.Name)
	// if err != nil {
	// 	return ctx, err
	// }

	// logger.Printf("%s normalised to %s", br.Name, dn)
	// _, auth, ok := br.Auth.Chosen()
	// if !ok {
	// 	return ctx, fmt.Errorf("no choice made for bind request auth")
	// }

	// var resp LdapMsg
	// switch auth.(type) {
	// case *string:
	// 	logger.Print("using simple auth")
	// 	newCtx, err := getAuthenticatedContext(ctx, s, dn)

	// 	var nfErr *d.NodeNotFoundError
	// 	if errors.As(err, &nfErr) {
	// 		diag := fmt.Sprintf("no object found for dn: %s", br.Name)
	// 		resp = NewBindResponse(msg.MessageId, NoSuchObject, nfErr.MatchedDN.String(), diag)
	// 		break
	// 	} else if err != nil {
	// 		return ctx, err
	// 	}

	// 	ctx = newCtx
	// 	diag := "success! your password was not checked yet but you're cool"
	// 	resp = NewBindResponse(msg.MessageId, Success, br.Name, diag)

	// case *SaslAuth:
	// 	logger.Print("using sasl auth")
	// 	resp = NewBindResponse(msg.MessageId, AuthMethodNotSupported, br.Name, "sasl auth unsupported")

	// default:
	// 	logger.Print("unknown auth type")
	// 	resp = NewBindResponse(msg.MessageId, ProtocolError, br.Name, "unknown bind auth type")
	// }

	///////
	diag := "success! your password was not checked yet but you're cool"
	resp := NewBindResponse(msg.MessageId, Success, br.Name, diag)
	//////

	var buf bytes.Buffer
	logger.Print("encoding...")
	if _, err := ber.Encode(&buf, resp); err != nil {
		return ctx, err
	}
	logger.Printf("... enc buff len is: %d bytes", buf.Len())
	if _, err := w.Write(buf.Bytes()); err != nil {
		return ctx, err
	}

	logger.Printf("%t", resp)

	return ctx, nil
}

var UnbindError = errors.New("unbind request recieved")

func HandleUnbindRequest(ctx context.Context, w io.Writer, s *Scheduler, msg LdapMsg) (context.Context, error) {
	return ctx, UnbindError
}
