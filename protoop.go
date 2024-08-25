package main

import (
	"fmt"

	asn1 "gopkg.in/asn1-ber.v1"
)

type ProtocolOp interface {
	PacketEncoder
	PacketDecoder
}

type ProtocolOpTag int

const (
	BindRequestTag  ProtocolOpTag = 0
	BindResponseTag ProtocolOpTag = 1
	AddRequestTag   ProtocolOpTag = 8
	AddResponseTag  ProtocolOpTag = 9
)

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

	err = protoOp.DecodePacket(p)

	return protoOp, nil
}

type BindRequest struct {
	version int64
	name    string
	simple  string
}

type AddRequest struct {
	Entry      string
	Attributes map[string]map[string]bool
}

type BindResponse Result
type AddResponse Result
