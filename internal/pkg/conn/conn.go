package conn

import (
	"bytes"
	"io"
	"net"

	asn1 "github.com/go-asn1-ber/asn1-ber"
)

type PacketEncoder interface {
	EncodePacket() *asn1.Packet
}

type LdapConn struct {
	conn   net.Conn
	closed bool
}

func NewLdapServer(conn net.Conn) *LdapConn {
	return &LdapConn{conn: conn}
}

func (c *LdapConn) Close() error {
	c.closed = true
	return nil
}

func (c *LdapConn) ReadPacket() (*asn1.Packet, error) {
	return asn1.ReadPacket(c.conn)
}

func (c *LdapConn) SendPacket(pe PacketEncoder) error {
	p := pe.EncodePacket()
	_, err := io.Copy(c.conn, bytes.NewReader(p.Bytes()))
	return err
}
