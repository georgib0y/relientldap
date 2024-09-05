package main

import (
	"log"
	"net"
)

func main() {
	entryRepo := PopulatedEntryRepo()
	schemaRepo := PopulatedSchemaRepo()

	schemaService := NewSchemaService(schemaRepo)
	entryService := NewEntryService(schemaService, entryRepo)

	controller := NewController(entryService)

	ln, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal(err)
	}

	tcpConn, err := ln.Accept()
	if err != nil {
		log.Fatal(err)
	}

	conn := NewConn(tcpConn)
	defer conn.Close()

	for {
		m, err := conn.ReadMessage()
		if err != nil {
			log.Fatal(err)

		}

		if _, ok := m.ProtocolOp.(*UnbindRequest); ok {
			log.Println("Unbind request recieved, closing connection")
			return
		}

		res, err := HandleMessage(controller, m)
		if err != nil {
			log.Fatal(err)
		}

		if err = conn.Send(res); err != nil {
			log.Fatal(err)
		}
	}
}

func PopulatedEntryRepo() EntryRepo {
	repo := NewMemEntryRepo()

	entries := []Entry{
		{
			id:         2,
			objClasses: map[OID]bool{},
			attrs: map[OID]map[string]bool{
				"dc-oid": {
					"georgiboy": true,
				},
			},
			parent:   ROOT_ID,
			children: map[ID]bool{},
		},

		{
			id:         ROOT_ID,
			objClasses: map[OID]bool{},
			attrs: map[OID]map[string]bool{
				"dc-oid": {
					"dev": true,
				},
			},
			parent: ROOT_ID,
			children: map[ID]bool{
				ID(2): true,
			},
		},
	}

	for _, entry := range entries {
		repo.Save(entry)
	}

	return repo
}

func PopulatedSchemaRepo() SchemaRepo {
	repo := NewMemSchemaRepo()

	repo.objClasses = map[OID]ObjectClass{
		"person-oid": {
			numericoid: "person-oid",
			names:      map[string]bool{"person": true},
			supOids:    map[OID]bool{"top": true},
			kind:       Structural,
			mustAttrs:  map[OID]bool{"cn-oid": true},
			mayAttrs:   map[OID]bool{"sn-oid": true},
		},
	}

	repo.attribues = map[OID]Attribute{
		"dc-oid": {
			numericoid: "dc-oid",
			names:      map[string]bool{"dc": true},
		},
		"cn-oid": {
			numericoid: "cn-oid",
			names:      map[string]bool{"cn": true, "commonName": true},
		},
		"sn-oid": {
			numericoid: "sn-oid",
			names:      map[string]bool{"sn": true, "surname": true},
		},
	}

	return repo
}
