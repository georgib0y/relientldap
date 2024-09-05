package main

import (
	"log"
	"strings"

	asn1 "github.com/go-asn1-ber/asn1-ber"
)

func logPacket(p *asn1.Packet) {
	var sb strings.Builder
	asn1.WritePacket(&sb, p)
	log.Println(sb.String())
}
