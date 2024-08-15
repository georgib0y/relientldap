package main

import (
	"fmt"
	"log"

	asn1 "gopkg.in/asn1-ber.v1"
)

type ProtocolOpTag int

const (
	BindRequestTag  ProtocolOpTag = 0
	BindResponseTag ProtocolOpTag = 1
	AddRequestTag   ProtocolOpTag = 8
)

type ProtocolOp interface {
	DecodeFromPacket(p *asn1.Packet) error
	EncodeAsPacket() *asn1.Packet
}

func protocolOpFromTag(t asn1.Tag) (ProtocolOp, error) {
	switch ProtocolOpTag(t) {
	case BindRequestTag:
		return &BindRequest{}, nil
	case AddRequestTag:
		return &AddRequest{}, nil
	}

	return nil, fmt.Errorf("Unknown protocol tag: %d", t)
}

func decodeProtocolOp(p *asn1.Packet) (ProtocolOp, error) {
	protoOp, err := protocolOpFromTag(p.Tag)
	if err != nil {
		return nil, err
	}

	err = protoOp.DecodeFromPacket(p)

	return protoOp, nil
}

type BindRequest struct {
	version int64
	name    string
	simple  string
}

func (b *BindRequest) DecodeFromPacket(p *asn1.Packet) error {
	if len(p.Children) != 3 {
		return fmt.Errorf("Packet has inccorrect number of children")
	}

	v, ok := p.Children[0].Value.(int64)
	if !ok {
		return fmt.Errorf("bind request version not an integer")
	}
	name, ok := p.Children[1].Value.(string)
	if !ok {
		return fmt.Errorf("bind request dn not a string")
	}

	authChoice := p.Children[2]
	if authChoice.Tag != 0 {
		return fmt.Errorf("unsupported authentication choice (simple only)")
	}

	// TODO should this be the way to do this??
	simple := string(authChoice.Data.Bytes())

	b.version = v
	b.name = name
	b.simple = string(simple)

	return nil
}

func (b *BindRequest) EncodeAsPacket() *asn1.Packet {
	return nil
}

type BindResponse struct {
	Result
}

func (b *BindResponse) DecodeFromPacket(p *asn1.Packet) error {
	return fmt.Errorf("decoding not implemented for response type")
}

func (b *BindResponse) EncodeAsPacket() *asn1.Packet {
	p := asn1.Encode(asn1.ClassApplication, asn1.TypeConstructed, asn1.Tag(BindResponseTag), nil, "BindResult")

	p.AppendChild(asn1.NewInteger(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagEnumerated, int32(b.ResultCode), "ResultCode"))
	p.AppendChild(asn1.NewString(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagOctetString, b.MatchedDN, "MatchedDN"))
	p.AppendChild(asn1.NewString(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagOctetString, b.DiagnosticMessage, "DiagnosticMessage"))

	return p
}

type AddRequest struct {
	Entry      string
	Attributes map[string][]string
}

func (a *AddRequest) DecodeFromPacket(p *asn1.Packet) error {
	if len(p.Children) < 1 {
		return fmt.Errorf("Packet has inccorrect number of children")
	}

	dn, ok := p.Children[0].Value.(string)
	if !ok {
		return fmt.Errorf("dn not a string")
	}

	attributes := map[string][]string{}
	for _, attrPkt := range p.Children[1:] {
		key, ok := attrPkt.Value.(string)
		if !ok {
			return fmt.Errorf("an attribute key is not a string")
		}

		vals := []string{}
		for _, valPkt := range attrPkt.Children {
			val, ok := valPkt.Value.(string)
			if !ok {
				return fmt.Errorf("an attribute value is not a string")
			}
			vals = append(vals, val)
		}
		attributes[key] = vals
	}

	a.Entry = dn
	a.Attributes = attributes

	return nil
}

func (a *AddRequest) EncodeAsPacket() *asn1.Packet {
	return nil
}

type Message struct {
	MessageId  int64
	ProtocolOp ProtocolOp
}

func decodeMessage(p *asn1.Packet) (Message, error) {
	if len(p.Children) < 2 {
		return Message{}, fmt.Errorf("Packet does not contain enough children")
	}

	msgId, ok := p.Children[0].Value.(int64)
	if !ok {
		return Message{}, fmt.Errorf("message id not an integer")
	}

	protoOp, err := decodeProtocolOp(p.Children[1])
	if err != nil {
		return Message{}, fmt.Errorf("Could not parse protocolop: %s", err)
	}

	if len(p.Children) > 2 {
		log.Println("Message likey contains some controls, but controls are yet to be implemented")
	}

	return Message{msgId, protoOp}, nil
}

func (m Message) encodeMessageAsPacket() *asn1.Packet {
	p := asn1.Encode(asn1.ClassUniversal, asn1.TypeConstructed, asn1.TagSequence, nil, "LDAPMessage")
	p.AppendChild(asn1.NewInteger(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagInteger, m.MessageId, "messageID"))
	p.AppendChild(m.ProtocolOp.EncodeAsPacket())

	return p
}

type ResultCode int32

const (
	ResultSuccess ResultCode = iota
)

type Result struct {
	ResultCode        ResultCode
	MatchedDN         string
	DiagnosticMessage string
	// todo referral
}
