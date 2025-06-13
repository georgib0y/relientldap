package conn

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"

	m "github.com/georgib0y/relientldap/internal/model"
	asn1 "github.com/go-asn1-ber/asn1-ber"
)

const (
	SimpleAuthChoice asn1.Tag = 0
)

var (
	UnsupportedAuthMethod = fmt.Errorf("unsupported authentication method")
)

type BindRequest struct {
	version      int64
	name, simple string
}

func NewBindRequest(p *asn1.Packet) (BindRequest, error) {
	if len(p.Children) != 3 {
		logger.Printf("expected 3 children got %d", len(p.Children))
		return BindRequest{}, InvalidPacket
	}

	v, ok := p.Children[0].Value.(int64)
	if !ok {
		logger.Printf("version not an int")
		return BindRequest{}, InvalidPacket
	}

	name, ok := p.Children[1].Value.(string)
	if !ok {
		logger.Printf("name not a string")
		return BindRequest{}, InvalidPacket
	}

	authp := p.Children[2]
	// TODO handle other auth methods
	switch authp.Tag {
	case SimpleAuthChoice:
		// not sure why asn1 has no value in Value but does in the Data buffer
		password := string(authp.Data.AvailableBuffer())
		return BindRequest{v, name, password}, nil
	default:
		logger.Printf("unknown auth tag: %d", authp.Tag)
		return BindRequest{}, UnsupportedAuthMethod
	}
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

func HandleBindRequest(ctx context.Context, conn net.Conn, ds *DitScheduler, p *asn1.Packet) (context.Context, error) {

	br, err := NewBindRequest(p.Children[1])
	if err != nil {
		return ctx, err
	}

	dn, err := m.NormaliseDN(ds.s, br.name)
	if err != nil {
		return ctx, err
	}

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
