package add

import (
	"fmt"
	"log"

	"github.com/georgib0y/gbldap/internal/pkg/conn"
	msg "github.com/georgib0y/gbldap/internal/pkg/message"
	asn1 "github.com/go-asn1-ber/asn1-ber"
)

type AddRequestHandler struct {
}

func (a *AddRequestHandler) Handle(c *conn.LdapConn, p *asn1.Packet) error {
	m, err := msg.NewMessage(p, arFromPacket)
	if err != nil {
		return err
	}

	log.Panicln("unimplemented AddEntry")
	// if _, err = AddEntry(m.ProtocolOp); err != nil {
	// 	return err
	// }

	res := msg.Message[AddResponse]{
		MessageId: m.MessageId,
		ProtocolOp: AddResponse{
			ResultCode: msg.Success,
		},
	}

	return c.SendPacket(res)
}

type AddRequest struct {
	Entry      string
	Attributes map[string]map[string]bool
}

func (a AddRequest) EncodePacket() *asn1.Packet {
	return nil
}

func parseAttribute(p *asn1.Packet) (string, map[string]bool, error) {
	if len(p.Children) != 2 {
		return "", nil, fmt.Errorf("Attribute packet has wrong number of children")
	}

	desc, ok := p.Children[0].Value.(string)
	if !ok {
		return "", nil, fmt.Errorf("an attribute desc is not a string")
	}

	vals := map[string]bool{}
	for _, p := range p.Children[1].Children {
		val, ok := p.Value.(string)
		if !ok {
			return "", nil, fmt.Errorf("an attribute value is not a string")
		}

		vals[val] = true
	}

	return desc, vals, nil
}

func arFromPacket(p *asn1.Packet) (AddRequest, error) {
	if len(p.Children) != 2 {
		return AddRequest{}, fmt.Errorf("Packet has inccorrect number of children")
	}

	dn, ok := p.Children[0].Value.(string)
	if !ok {
		return AddRequest{}, fmt.Errorf("dn not a string")
	}

	attributes := map[string]map[string]bool{}
	for _, attr := range p.Children[1].Children {
		desc, vals, err := parseAttribute(attr)
		if err != nil {

			return AddRequest{}, err
		}

		attributes[desc] = vals
	}

	return AddRequest{dn, attributes}, nil
}

type AddResponse msg.Result

func (a AddResponse) EncodePacket() *asn1.Packet {
	return msg.EncodeResult((msg.Result)(a), asn1.Tag(msg.BindResponseTag), "AddResponse")
}
