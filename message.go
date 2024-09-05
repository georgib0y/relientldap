package main

import (
	"fmt"
	"log"

	asn1 "github.com/go-asn1-ber/asn1-ber"
)

type PacketEncoder interface {
	EncodePacket() *asn1.Packet
}

type PacketDecoder interface {
	DecodePacket(p *asn1.Packet) error
}

type ProtocolOp interface {
	PacketEncoder
	PacketDecoder
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

func protocolOpFromTag(t asn1.Tag) (ProtocolOp, error) {
	switch ProtocolOpTag(t) {
	case BindRequestTag:
		return &BindRequest{}, nil
	case UnbindRequestTag:
		return &UnbindRequest{}, nil
	case SearchRequestTag:
		return &SearchRequest{}, nil
	case AddRequestTag:
		return &AddRequest{}, nil
	}

	return nil, fmt.Errorf("Unknown protocol tag: %d", t)
}

func decodeProtocolOp(p *asn1.Packet) (ProtocolOp, error) {
	protoOp, err := protocolOpFromTag(p.Tag)
	if err != nil {
		logPacket(p)
		return nil, err
	}

	err = protoOp.DecodePacket(p)

	return protoOp, err
}

type BindRequest struct {
	version int64
	name    string
	simple  string
}

func (b *BindRequest) EncodePacket() *asn1.Packet {
	return nil
}

func (b *BindRequest) DecodePacket(p *asn1.Packet) error {
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

	simple, ok := authChoice.Value.(string)
	// if no password was provided
	if !ok && authChoice.Value == nil {
		log.Println("no password was provided")
		simple = ""
	} else if !ok {
		return fmt.Errorf("Simple auth data is %T and not a string", authChoice.Value)
	}

	b.version = v
	b.name = name
	b.simple = string(simple)

	return nil
}

type BindResponse Result

func (b *BindResponse) EncodePacket() *asn1.Packet {
	return encodeResult((*Result)(b), asn1.Tag(BindResponseTag), "BindResponse")
}

func (b *BindResponse) DecodePacket(p *asn1.Packet) error {
	return fmt.Errorf("BindResponse decoding unimplemented")
}

type UnbindRequest struct{}

func (u *UnbindRequest) EncodePacket() *asn1.Packet {
	return nil
}

func (u *UnbindRequest) DecodePacket(p *asn1.Packet) error {
	// the packet should be null so do nothing
	return nil
}

type AddRequest struct {
	Entry      string
	Attributes map[string]map[string]bool
}

func (a *AddRequest) EncodePacket() *asn1.Packet {
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

func (a *AddRequest) DecodePacket(p *asn1.Packet) error {
	if len(p.Children) != 2 {
		return fmt.Errorf("Packet has inccorrect number of children")
	}

	logPacket(p)

	dn, ok := p.Children[0].Value.(string)
	if !ok {
		return fmt.Errorf("dn not a string")
	}

	attributes := map[string]map[string]bool{}
	for _, attr := range p.Children[1].Children {
		desc, vals, err := parseAttribute(attr)
		if err != nil {

			return err
		}

		attributes[desc] = vals
	}

	a.Entry = dn
	a.Attributes = attributes

	log.Printf("a entry is: %s", a.Entry)

	return nil
}

type AddResponse Result

func (a *AddResponse) EncodePacket() *asn1.Packet {
	return encodeResult((*Result)(a), asn1.Tag(BindResponseTag), "AddResponse")
}

func (a *AddResponse) DecodePacket(p *asn1.Packet) error {
	return fmt.Errorf("AddResponse decoding unimplemented")
}

type SearchScope int

const (
	BaseObject   SearchScope = 0
	SingleLevel  SearchScope = 1
	WholeSubtree SearchScope = 2
)

type SearchRequest struct {
	baseObject string
	scope      SearchScope
	// TODO deref alias
	sizeLim, timeLim int
	typesOnly        bool
	// TODO Filter
	// TODO Attributes
}

func (s *SearchRequest) EncodePacket() *asn1.Packet {
	return nil
}

func (s *SearchRequest) DecodePacket(p *asn1.Packet) error {
	if len(p.Children) != 8 {
		return fmt.Errorf("Packet has inccorrect number of children")
	}

	dn, ok := p.Children[0].Value.(string)
	if !ok {
		return fmt.Errorf("dn not a string")
	}

	scope, ok := p.Children[1].Value.(SearchScope)
	if !ok || scope > WholeSubtree {
		return fmt.Errorf("invalid scope")
	}

	// TODO Deref alias

	sizeLim, ok := p.Children[3].Value.(int)
	if !ok {
		return fmt.Errorf("invalid sizeLim")
	}

	timeLim, ok := p.Children[4].Value.(int)
	if !ok {
		return fmt.Errorf("invalid timeLim")
	}

	typesOnly, ok := p.Children[5].Value.(bool)
	if !ok {
		return fmt.Errorf("invalid typeOnly")
	}

	// TODO Filter

	// TODO Attributes

	s.baseObject = dn
	s.scope = scope
	s.sizeLim = sizeLim
	s.timeLim = timeLim
	s.typesOnly = typesOnly

	return nil
}

type SearchResultDoneResponse Result

func (s *SearchResultDoneResponse) EncodePacket() *asn1.Packet {
	return encodeResult((*Result)(s), asn1.Tag(asn1.Tag(SearchResultDoneResponseTag)), "SearchResultDoneResponse")
}

func (s *SearchResultDoneResponse) DecodePacket(p *asn1.Packet) error {
	return fmt.Errorf("AddResponse decoding unimplemented")
}
