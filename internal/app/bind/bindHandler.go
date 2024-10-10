package bind

import (
	"fmt"
	"log"

	"github.com/georgib0y/gbldap/internal/pkg/conn"
	msg "github.com/georgib0y/gbldap/internal/pkg/message"
	asn1 "github.com/go-asn1-ber/asn1-ber"
)

type BindRequest struct {
	version int64
	name    string
	simple  string
}

func (b BindRequest) EncodePacket() *asn1.Packet {
	return nil
}

func brFromPacket(p *asn1.Packet) (BindRequest, error) {
	if len(p.Children) != 3 {
		return BindRequest{}, fmt.Errorf("Packet has inccorrect number of children")
	}

	v, ok := p.Children[0].Value.(int64)
	if !ok {
		return BindRequest{}, fmt.Errorf("bind request version not an integer")
	}
	name, ok := p.Children[1].Value.(string)
	if !ok {
		return BindRequest{}, fmt.Errorf("bind request dn not a string")
	}

	authChoice := p.Children[2]
	if authChoice.Tag != 0 {
		return BindRequest{}, fmt.Errorf("unsupported authentication choice (simple only)")
	}

	simple, ok := authChoice.Value.(string)
	// if no password was provided
	if !ok && authChoice.Value == nil {
		log.Println("no password was provided")
		simple = ""
	} else if !ok {
		return BindRequest{}, fmt.Errorf("Simple auth data is %T and not a string", authChoice.Value)
	}

	return BindRequest{v, name, string(simple)}, nil
}

type BindResponse msg.Result

func (b BindResponse) EncodePacket() *asn1.Packet {
	return msg.EncodeResult((msg.Result)(b), asn1.Tag(msg.BindResponseTag), "BindResponse")
}

func HandleBindRequest(c *conn.LdapConn, p *asn1.Packet) error {
	m, err := msg.NewMessage(p, brFromPacket)
	if err != nil {
		return err
	}

	res := msg.Message[BindResponse]{
		MessageId: m.MessageId,
		ProtocolOp: BindResponse{
			ResultCode:        msg.Success,
			MatchedDN:         m.ProtocolOp.name,
			DiagnosticMessage: "Passwords not implemented yet - you seem trustworthy though",
		},
	}

	return c.SendPacket(res)
}
