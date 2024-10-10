package conn

import (
	"fmt"

	asn1 "github.com/go-asn1-ber/asn1-ber"
)

type Handler interface {
	Handle(c *LdapConn, p *asn1.Packet) error
}

type Mux struct {
	handlers map[asn1.Tag]Handler
}

func NewMux() *Mux {
	return &Mux{map[asn1.Tag]Handler{}}
}

func (m *Mux) AddHandler(tag asn1.Tag, h Handler) {
	m.handlers[tag] = h
}

func (m *Mux) ServeConn(c *LdapConn) error {
	defer c.Close()

	for !c.closed {
		p, err := c.ReadPacket()
		if err != nil {
			return err
		}

		if len(p.Children) < 2 {
			return fmt.Errorf("router: could not get protoop tag, not enough children")
		}

		tag := p.Children[1].Tag
		h, ok := m.handlers[tag]
		if !ok {
			return fmt.Errorf("router: unknown protoop tag %d", tag)
		}

		if err := h.Handle(c, p); err != nil {
			return err
		}
	}

	return nil
}
