package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	d "github.com/georgib0y/relientldap/internal/domain"
	"github.com/georgib0y/relientldap/internal/util"
	"github.com/georgib0y/relientldap/pkg/ber"
)

type ContextKey int

const (
	BoundEntryKey ContextKey = iota
)

var logger = log.New(os.Stderr, "server: ", log.Lshortfile)

var (
	InvalidPacket = fmt.Errorf("Invalid Packet")
)

type Handler interface {
	RequestTag() ber.Tag
	ResponseTag() ber.Tag
	Handle(ctx context.Context, w io.Writer, msg LdapMsg) error
}

type handleFunc struct {
	reqTag ber.Tag
	resTag ber.Tag
	handle func(context.Context, io.Writer, LdapMsg) error
}

func (h *handleFunc) RequestTag() ber.Tag {
	return h.reqTag
}

func (h *handleFunc) ResponseTag() ber.Tag {
	return h.resTag
}

func (h *handleFunc) Handle(ctx context.Context, w io.Writer, msg LdapMsg) error {
	return h.handle(ctx, w, msg)
}

func HandleFunc(reqTag, resTag ber.Tag, handle func(context.Context, io.Writer, LdapMsg) error) Handler {
	return &handleFunc{reqTag, resTag, handle}
}

type Mux struct {
	handlers map[ber.Tag]Handler
}

func NewMux() *Mux {
	return &Mux{map[ber.Tag]Handler{}}
}

func (m *Mux) AddHandler(h Handler) *Mux {
	m.handlers[h.RequestTag()] = h
	return m
}

func writeResponse(w io.Writer, res LdapMsg) error {
	if res == (LdapMsg{}) {
		return fmt.Errorf("trying to write an empty response - not encoding!")
	}

	var buf bytes.Buffer
	logger.Printf("encoding %v...", res)
	_, err := ber.Encode(&buf, res)
	if err != nil {
		return err
	}
	logger.Printf("... enc buff len is: %d bytes", buf.Len())
	_, err = w.Write(buf.Bytes())
	if err != nil {
		return err
	}
	logger.Printf("encoded %v", res)
	return nil
}

func tryWriteErr(h Handler, w io.Writer, msgId int, err error) error {
	lerr, ok := err.(d.LdapError)
	if !ok {
		return err
	}

	res := NewResultMsg(
		h.ResponseTag(),
		msgId,
		lerr.ResultCode,
		lerr.MatchedDN.String(),
		"%s", lerr.DiagnosticMessage,
	)
	return writeResponse(w, res)
}

func (m *Mux) Serve(c net.Conn) {
	defer c.Close()

	r := io.TeeReader(c, util.NewHexLogger("in"))
	w := io.MultiWriter(util.NewHexLogger("out"), c)

	boundEntry := new(*d.Entry)
	ctx := context.WithValue(context.Background(), BoundEntryKey, boundEntry)

	for {
		logger.Print("recieving message...")
		var msg LdapMsg
		if err := ber.Decode(r, &msg); err != nil {
			logger.Print(err)
			return
		}

		logger.Print("decoded message")

		tag, _, ok := msg.Request.Chosen()
		if !ok {
			logger.Printf("no choices was made for incoming ldap message")
			return
		}
		h, ok := m.handlers[tag]
		if !ok {
			logger.Printf("unkown ldapmsg tag %s", tag)
			return
		}

		err := h.Handle(ctx, w, msg)
		if errors.Is(err, UnbindError) {
			logger.Print("recieved unbind request, closing connection")
			return
		} else if err != nil {
			err = tryWriteErr(h, w, msg.MessageId, err)
			if err != nil {
				logger.Printf("unrecoverable err: %s", err)
				return
			}
		}

		logger.Print("... sent response")

	}
}
