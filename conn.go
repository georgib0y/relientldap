package main

import (
	"bytes"
	"io"
	"net"

	asn1 "gopkg.in/asn1-ber.v1"
)

type Conn struct {
	conn net.Conn
}

func NewConn(conn net.Conn) *Conn {
	return &Conn{conn}
}

func (c *Conn) ReadMessage() (Message, error) {
	p, err := asn1.ReadPacket(c.conn)
	if err != nil {
		return Message{}, err
	}

	msg, err := decodeMessage(p)
	if err != nil {
		return Message{}, err
	}

	return msg, err
}

func (c *Conn) Send(e PacketEncoder) error {
	p := e.EncodePacket()
	if _, err := io.Copy(c.conn, bytes.NewReader(p.Bytes())); err != nil {
		return err
	}

	return nil
}

func (c *Conn) Close() error {
	return c.conn.Close()
}
