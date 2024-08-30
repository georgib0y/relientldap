package main

import (
	"fmt"

	asn1 "github.com/go-asn1-ber/asn1-ber"
)

type PacketDecoder interface {
	DecodePacket(p *asn1.Packet) error
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

	// TODO should this be the way to do this??
	simple := string(authChoice.Data.Bytes())

	b.version = v
	b.name = name
	b.simple = string(simple)

	return nil
}

func (b *BindResponse) DecodePacket(p *asn1.Packet) error {
	return fmt.Errorf("BindResponse decoding unimplemented")
}

func (u *UnbindRequest) DecodePacket(p *asn1.Packet) error {
	// the packet should be null so do nothing
	return nil
}

func (a *AddRequest) DecodePacket(p *asn1.Packet) error {
	if len(p.Children) < 1 {
		return fmt.Errorf("Packet has inccorrect number of children")
	}

	dn, ok := p.Children[0].Value.(string)
	if !ok {
		return fmt.Errorf("dn not a string")
	}

	attributes := map[string]map[string]bool{}
	for _, attrPkt := range p.Children[1:] {
		desc, vals, err := parseAttribute(attrPkt)
		if err != nil {
			return err
		}

		attributes[desc] = vals
	}

	a.Entry = dn
	a.Attributes = attributes

	return nil
}

func parseAttribute(p *asn1.Packet) (string, map[string]bool, error) {
	desc, ok := p.Value.(string)
	if !ok {
		return "", nil, fmt.Errorf("an attribute desc is not a string")
	}

	vals := map[string]bool{}
	for _, valPkt := range p.Children {
		val, ok := valPkt.Value.(string)
		if !ok {
			return "", nil, fmt.Errorf("an attribute value is not a string")
		}

		vals[val] = true
	}

	return desc, vals, nil
}

func (a *AddResponse) DecodePacket(p *asn1.Packet) error {
	return fmt.Errorf("AddResponse decoding unimplemented")
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

func (s *SearchResultDoneResponse) DecodePacket(p *asn1.Packet) error {
	return fmt.Errorf("AddResponse decoding unimplemented")
}
