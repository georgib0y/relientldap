package main

import (
	"fmt"

	asn1 "gopkg.in/asn1-ber.v1"
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
