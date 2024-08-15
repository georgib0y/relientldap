package main

import (
	"bytes"
	"io"
	"net"

	asn1 "gopkg.in/asn1-ber.v1"
)

type Conn struct {
	conn     net.Conn
	messages chan Message
	errors   chan error
}

func NewConn(conn net.Conn, messages chan Message, errors chan error) *Conn {
	return &Conn{conn, messages, errors}
}

func (c *Conn) ReadMessage() {
	p, err := asn1.ReadPacket(c.conn)
	if err != nil {
		c.errors <- err
		return
	}

	msg, err := decodeMessage(p)
	if err != nil {
		c.errors <- err
		return
	}

	c.messages <- msg
}

func (c *Conn) SendMessage(m Message) {
	p := m.encodeMessageAsPacket()
	if _, err := io.Copy(c.conn, bytes.NewReader(p.Bytes())); err != nil {
		c.errors <- err
	}

}

func (c *Conn) Messages() chan Message {
	return c.messages
}

func (c *Conn) Errors() chan error {
	return c.errors
}
