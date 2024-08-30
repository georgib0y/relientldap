package main

import (
	"fmt"
	"log"
	"strings"

	asn1 "github.com/go-asn1-ber/asn1-ber"
)

type ProtocolOp interface {
	PacketEncoder
	PacketDecoder
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
		var sb strings.Builder
		asn1.WritePacket(&sb, p)
		log.Println(sb.String())
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

type BindResponse Result

type UnbindRequest struct{}

type AddRequest struct {
	Entry      string
	Attributes map[string]map[string]bool
}

type AddResponse Result

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

type SearchResultDoneResponse Result
