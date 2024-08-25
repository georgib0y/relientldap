package main

import (
	asn1 "gopkg.in/asn1-ber.v1"
)

type PacketEncoder interface {
	EncodePacket() *asn1.Packet
}

func (b *BindRequest) EncodePacket() *asn1.Packet {
	return nil
}

func (b *BindResponse) EncodePacket() *asn1.Packet {
	return encodeResult((*Result)(b), asn1.Tag(BindResponseTag), "BindResponse")
}

func (a *AddRequest) EncodePacket() *asn1.Packet {
	return nil
}

func (a *AddResponse) EncodePacket() *asn1.Packet {
	return encodeResult((*Result)(a), asn1.Tag(BindResponseTag), "AddResponse")
}
