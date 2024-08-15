package main

import (
	"bytes"
	"io"
	"log"
	"net"

	asn1 "gopkg.in/asn1-ber.v1"
)

func main() {
	ln, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal(err)
	}

	conn, err := ln.Accept()
	if err != nil {
		log.Fatal(err)
	}

	defer conn.Close()

	for {
		p, err := asn1.ReadPacket(conn)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("print packet 1: %v", *p)
		asn1.PrintPacket(p)

		msg, err := decodeMessage(p)
		if err != nil {
			log.Fatal(err)
		}

		bindReq, ok := msg.ProtocolOp.(*BindRequest)
		if !ok {
			log.Println("print packet 3")
			asn1.PrintPacket(p)
			log.Fatal("Msg not bind request")
		}

		bindRes := BindResponse{Result{
			ResultCode:        ResultSuccess,
			MatchedDN:         bindReq.name,
			DiagnosticMessage: "Hello from go",
		}}

		resp := Message{
			MessageId:  1,
			ProtocolOp: &bindRes,
		}

		p = resp.encodeMessageAsPacket()
		log.Println("print packet 3")
		asn1.PrintPacket(p)

		if _, err := io.Copy(conn, bytes.NewBuffer(p.Bytes())); err != nil {
			log.Fatal(err)
		}
		log.Println("wrote packet")
	}
}
