package conn

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"

	m "github.com/georgib0y/relientldap/internal/model"
)

type ContextKey int

const (
	BoundDnKey ContextKey = iota
)

var (
	logger = log.New(os.Stderr, "conn: ", log.Lshortfile)
)

var (
	InvalidPacket = fmt.Errorf("Invalid Packet")
)

type Action func(m.DIT)

type DitScheduler struct {
	d     m.DIT
	s     *m.Schema
	queue chan Action
}

func NewDitScheduler(d m.DIT, s *m.Schema) *DitScheduler {
	return &DitScheduler{d: d, s: s, queue: make(chan Action)}
}

func (s *DitScheduler) Run() {
	for a := range s.queue {
		a(s.d)
	}
}

func (s *DitScheduler) Schedule(action Action) {
	s.queue <- action
}

type HandleFunc = func(ctx context.Context, conn net.Conn, ds *DitScheduler, p *asn1.Packet) (context.Context, error)

type Mux struct {
	scheduler *DitScheduler
	handlers  map[asn1.Tag]HandleFunc
}

func NewMux(scheduler *DitScheduler) *Mux {
	return &Mux{scheduler, map[asn1.Tag]HandleFunc{}}
}

func (m *Mux) AddHandler(tag asn1.Tag, h HandleFunc) *Mux {
	m.handlers[tag] = h
	return m
}

func (m *Mux) Serve(c net.Conn) {
	defer c.Close()

	// map of cancel funcs for any abandon requests that come through
	// unused atm but may be needed later
	cancels := map[int64]func(){}

	ctx := context.Background()

	for {
		logger.Print("recieving message...")
		p, err := asn1.ReadPacket(c)
		if err != nil {
			logger.Print(err)
			return
		}

		msgId, _, err := msgIdAndControls(p)
		if err != nil {
			logger.Print(err)
			return
		}

		currCtx, cancel := context.WithCancel(ctx)
		cancels[msgId] = cancel

		tag := p.Children[1].Tag
		handler, ok := m.handlers[tag]
		if !ok {
			logger.Printf("unkown protoop tag %d", tag)
			return
		}

		newCtx, err := handler(currCtx, c, m.scheduler, p)
		if err != nil {
			logger.Print(err)
			return
		}
		ctx = newCtx

		delete(cancels, msgId)
	}
}

// TODO implement control parssing
type Control = struct{}

// func msgIdAndControls(p *asn1.Packet) (int64, []Control, error) {
// 	if len(p.Children) < 2 {
// 		logger.Println("expected message envelope to have at least 2 children, got %d", len(p.Children))
// 		return 0, []Control{}, InvalidPacket
// 	}

// 	msgId, ok := p.Children[0].Value.(int64)
// 	if !ok {
// 		logger.Println("message id not an int")
// 		return 0, []Control{}, InvalidPacket
// 	}

// 	// TODO controls
// 	if len(p.Children) > 2 {
// 		logger.Println("message env likely has controls, but they aren't implemented yet")
// 	}

// 	return msgId, []Control{}, nil
// }

type ResultCode int

const (
	Success ResultCode = iota
)

type ResponseTag uint32

// const (
// 	BindResponse ResponseTag = 1
// )

// func NewResponsePacket(tag ResponseTag) *asn1.Packet {
// 	return asn1.Encode(
// 		asn1.ClassApplication,
// 		asn1.TypeConstructed,
// 		asn1.Tag(tag),
// 		nil,
// 		"Response",
// 	)
// }

// // TODO referral
// func PutLdapResult(p *asn1.Packet, code ResultCode, matchedDn string, diagnostic string) {
// 	p.AppendChild(asn1.NewInteger(
// 		asn1.ClassUniversal,
// 		asn1.TypePrimitive,
// 		asn1.TagEnumerated,
// 		code,
// 		"ResultCode",
// 	))

// 	p.AppendChild(asn1.NewString(
// 		asn1.ClassUniversal,
// 		asn1.TypePrimitive,
// 		asn1.TagOctetString,
// 		matchedDn,
// 		"MatchedDN",
// 	))

// 	p.AppendChild(asn1.NewString(
// 		asn1.ClassUniversal,
// 		asn1.TypePrimitive,
// 		asn1.TagOctetString,
// 		diagnostic,
// 		"DiagnosticMessage",
// 	))
// }
