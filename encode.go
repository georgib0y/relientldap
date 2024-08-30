package main

import (
	asn1 "github.com/go-asn1-ber/asn1-ber"
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

func (u *UnbindRequest) EncodePacket() *asn1.Packet {
	return nil
}

func (a *AddRequest) EncodePacket() *asn1.Packet {
	return nil
}

func (a *AddResponse) EncodePacket() *asn1.Packet {
	return encodeResult((*Result)(a), asn1.Tag(BindResponseTag), "AddResponse")
}

func (s *SearchRequest) EncodePacket() *asn1.Packet {
	return nil
}

func (s *SearchResultDoneResponse) EncodePacket() *asn1.Packet {
	return encodeResult((*Result)(s), asn1.Tag(asn1.Tag(SearchResultDoneResponseTag)), "SearchResultDoneResponse")
}
