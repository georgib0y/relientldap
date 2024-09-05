package main

import "testing"

func TestNormaliseDN(t *testing.T) {
	schemaRepo := PopulatedSchemaRepo()
	t.Log(schemaRepo)

	memSr := schemaRepo.(*MemSchemaRepo)
	t.Log(memSr)

	t.Log(memSr.objClasses)
	schemaService := NewSchemaService(schemaRepo)

	dnStr := "cn=Dexter McClary,dc=georgiboy,dc=dev"

	dn, err := schemaService.NormaliseDN(dnStr)
	if err != nil {
		t.Fatalf("%T: %s", err, err)
	}

	t.Log(dn)
}


