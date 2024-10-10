package message

import (
	"fmt"
	"log"

	"github.com/georgib0y/gbldap/internal/pkg/conn"
	asn1 "github.com/go-asn1-ber/asn1-ber"
)

type Message[T conn.PacketEncoder] struct {
	MessageId  int64
	ProtocolOp T
}

func NewMessage[T conn.PacketEncoder](p *asn1.Packet, protoOpDec func(p *asn1.Packet) (T, error)) (Message[T], error) {
	if len(p.Children) < 2 {
		return Message[T]{}, fmt.Errorf("Packet does not contain enough children")
	}

	msgId, ok := p.Children[0].Value.(int64)
	if !ok {
		return Message[T]{}, fmt.Errorf("message id not an integer")
	}

	protoOp, err := protoOpDec(p.Children[1])
	if err != nil {
		return Message[T]{}, fmt.Errorf("Could not parse protocolop: %s", err)
	}

	if len(p.Children) > 2 {
		log.Println("Message likey contains some controls, but controls are yet to be implemented")
	}

	return Message[T]{msgId, protoOp}, nil
}

func (m Message[T]) EncodePacket() *asn1.Packet {
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

func EncodeResult(r Result, tag asn1.Tag, desc string) *asn1.Packet {
	p := asn1.Encode(asn1.ClassApplication, asn1.TypeConstructed, tag, nil, desc)

	p.AppendChild(asn1.NewInteger(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagEnumerated, int32(r.ResultCode), "ResultCode"))
	p.AppendChild(asn1.NewString(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagOctetString, r.MatchedDN, "MatchedDN"))
	p.AppendChild(asn1.NewString(asn1.ClassUniversal, asn1.TypePrimitive, asn1.TagOctetString, r.DiagnosticMessage, "DiagnosticMessage"))

	return p
}

type ProtocolOpTag int

const (
	BindRequestTag              ProtocolOpTag = 0
	BindResponseTag             ProtocolOpTag = 1
	UnbindRequestTag            ProtocolOpTag = 2
	SearchRequestTag            ProtocolOpTag = 3
	SeachResultEntryResponseTag ProtocolOpTag = 4
	SearchResultDoneResponseTag ProtocolOpTag = 5
	AddRequestTag               ProtocolOpTag = 8
	AddResponseTag              ProtocolOpTag = 9
)
