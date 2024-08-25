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
			id: 2,
			attrs: map[OID]map[string]bool{
				"dc-oid": {
					"georgiboy": true,
				},
			},
		},

		{
			id: ROOT_ID,
			children: map[ID]bool{
				ID(2): true,
			},
			attrs: map[OID]map[string]bool{
				"dc-oid": {
					"dev": true,
				},
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
			mustAttrs:  map[OID]bool{"cn-oid": true},
			mayAttrs:   map[OID]bool{"sn-oid": true},
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
