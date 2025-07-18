package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/georgib0y/relientldap/internal/util"
	"github.com/georgib0y/relientldap/pkg/ber"
)

type ContextKey int

const (
	BoundDnKey ContextKey = iota
)

var logger = log.New(os.Stderr, "server: ", log.Lshortfile)

var (
	InvalidPacket = fmt.Errorf("Invalid Packet")
)

type HandleFunc = func(ctx context.Context, w io.Writer, s *Scheduler, msg LdapMsg) (context.Context, error)

type Mux struct {
	scheduler *Scheduler
	handlers  map[ber.Tag]HandleFunc
}

func NewMux(scheduler *Scheduler) *Mux {
	return &Mux{scheduler, map[ber.Tag]HandleFunc{}}
}

func (m *Mux) AddHandler(tag ber.Tag, h HandleFunc) *Mux {
	m.handlers[tag] = h
	return m
}

func (m *Mux) Serve(c net.Conn) {
	defer c.Close()

	teeIn := io.TeeReader(c, util.NewHexLogger(logger, "in"))
	teeOut := io.MultiWriter(util.NewHexLogger(logger, "out"), c)

	ctx := context.Background()

	for {
		logger.Print("recieving message...")
		var msg LdapMsg
		if err := ber.Decode(teeIn, &msg); err != nil {
			logger.Print(err)
			return
		}

		logger.Print("decoded message")

		tag, _, ok := msg.Request.Chosen()
		if !ok {
			logger.Printf("no choices was made for incoming ldap message")
			return
		}
		handler, ok := m.handlers[tag]
		if !ok {
			logger.Printf("unkown ldapmsg tag %s", tag)
			return
		}

		newCtx, err := handler(ctx, teeOut, m.scheduler, msg)
		if errors.Is(err, UnbindError) {
			logger.Print("recieved unbind request, closing connection")
			return
		} else if err != nil {
			// TODO handle handler errors more gracefully
			logger.Print(err)
			return
		}

		logger.Print("... sent response")
		ctx = newCtx
	}
}
