package bind

import (
	"github.com/georgib0y/relientldap/internal/pkg/conn"
	asn1 "github.com/go-asn1-ber/asn1-ber"
)

func HandleUnbindRequest(c *conn.LdapConn, _p *asn1.Packet) error {
	return c.Close()
}
