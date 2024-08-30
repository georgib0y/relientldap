package main

import (
	"fmt"
	"log"

	asn1 "github.com/go-asn1-ber/asn1-ber"
)

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

func (m Message) EncodePacket() *asn1.Packet {
	p := asn1.Encode(asn1.ClassUniversal, asn1.TypeConstructed, asn1.TagSequence, nil, "LDAPMessage")
	p.AppendChild(asn1.NewInteger(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagInteger, m.MessageId, "messageID"))
	p.AppendChild(m.ProtocolOp.EncodePacket())

	return p
}

type ResultCode int32

const (
	Success                ResultCode = 0
	UndefinedAttributeType ResultCode = 17
	NoSuchObject           ResultCode = 32
	InvalidDNSyntax        ResultCode = 34
	EntryAlreadyExists     ResultCode = 68
)

type Result struct {
	ResultCode        ResultCode
	MatchedDN         string
	DiagnosticMessage string
	// todo referral
}

func encodeResult(r *Result, tag asn1.Tag, desc string) *asn1.Packet {
	p := asn1.Encode(asn1.ClassApplication, asn1.TypeConstructed, tag, nil, desc)

	p.AppendChild(asn1.NewInteger(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagEnumerated, int32(r.ResultCode), "ResultCode"))
	p.AppendChild(asn1.NewString(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagOctetString, r.MatchedDN, "MatchedDN"))
	p.AppendChild(asn1.NewString(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagOctetString, r.DiagnosticMessage, "DiagnosticMessage"))

	return p
}
