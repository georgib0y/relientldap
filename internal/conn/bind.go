package conn

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"

	m "github.com/georgib0y/relientldap/internal/model"
	"github.com/georgib0y/relientldap/pkg/ber"
)

var (
	UnsupportedAuthMethod = fmt.Errorf("unsupported authentication method")
)

type BindRequest struct {
	Version int
	Name    string
	Auth    *BindReqChoice
}

type SaslAuth struct {
	Mechanism   string
	Credentials []byte
}

var (
	BrSimpleTag = ber.Tag{Class: ber.ContextSpecific, Construct: ber.Constructed, Value: 0}
	BrSaslTag   = ber.Tag{Class: ber.ContextSpecific, Construct: ber.Constructed, Value: 3}
	BindRespTag = ber.Tag{Class: ber.Application, Construct: ber.Constructed, Value: 1}
)

type BindReqChoice struct {
	t      *ber.Tag
	Simple string
	Sasl   SaslAuth
}

func NewSimpleBindRequest(simple string) *BindReqChoice {
	var br BindReqChoice
	br.t = new(ber.Tag)
	*br.t = BrSimpleTag
	br.Simple = simple
	return &br
}

func NewSaslBindRequest(mechanism string, credentials []byte) *BindReqChoice {
	var br BindReqChoice
	br.t = new(ber.Tag)
	*br.t = BrSaslTag
	br.Sasl = SaslAuth{mechanism, credentials}
	return &br
}

func (b *BindReqChoice) Choose(t ber.Tag) (any, error) {
	switch {
	case t.Equals(BrSimpleTag):
		b.t = new(ber.Tag)
		*b.t = t
		return &b.Simple, nil
	case t.Equals(BrSaslTag):
		b.t = new(ber.Tag)
		*b.t = t
		return &b.Sasl, nil
	}

	return nil, fmt.Errorf("unexpected tag for bind request choice")
}

func (b *BindReqChoice) Tag() (ber.Tag, bool) {
	if b.t == nil {
		return ber.Tag{}, false
	}
	return *b.t, true
}

type BindResponse struct {
	ResultCode        ResultCode
	MatchedDN         string
	diagnosticMessage string
}

func getAuthenticatedContext(ctx context.Context, ds *DitScheduler, dn m.DN) (context.Context, error) {
	done := make(chan context.Context)
	errChan := make(chan error)

	ds.Schedule(func(d m.DIT) {
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

func HandleBindRequest(ctx context.Context, conn net.Conn, ds *DitScheduler, br BindRequest) (context.Context, error) {

	dn, err := m.NormaliseDN(ds.s, br.Name)
	if err != nil {
		return ctx, err
	}

	// TODO extract simple/sasl choice

	newCtx, err := getAuthenticatedContext(ctx, ds, dn)
	if err != nil {
		return ctx, err
	}

	resp := NewResponsePacket(BindResponse)
	PutLdapResult(resp, Success, br.name, "success! your password was not checked yet but you're cool")
	r := bytes.NewReader(resp.Bytes())

	if _, err = io.Copy(conn, r); err != nil {
		return ctx, err
	}

	return newCtx, nil
}
