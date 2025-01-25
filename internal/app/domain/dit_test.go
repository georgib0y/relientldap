package domain

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

var test_dit DIT

/*
Test DIT Structue
dc=dev
dc=georgiboy
cn=Test1 | ou=TestOu
         | cn=Test2
*/

func generateTestDIT() DIT {	
	cn_test2 := DITNode {
		entry: Entry{
			attrs: map[OID]map[string]bool {
				"cn": map[string]bool{"Test2": true},
			},
		},
	}

	
	ou_test_ou := DITNode {
		children: []*DITNode{&cn_test2},
		entry: Entry{
			attrs: map[OID]map[string]bool {
				"ou": map[string]bool{"TestOu": true},
			},
		},
	}
	
	cn_test1 := DITNode {
		entry: Entry{
			attrs: map[OID]map[string]bool {
				"cn": map[string]bool{"Test1": true},
			},
		},
	}

	dc_georgiboy := DITNode { children: []*DITNode{&cn_test1, &ou_test_ou},
		entry: Entry{
			attrs: map[OID]map[string]bool {
				"dc": map[string]bool{"georgiboy": true},
			},
		},
	}

	dc_dev := DITNode {
		children: []*DITNode{&dc_georgiboy},
		entry: Entry{
			attrs: map[OID]map[string]bool {
				"dc": map[string]bool{"dev": true},
			},
		},
	}

	
	return DIT{&dc_dev}
}

func TestMain(m *testing.M) {
	test_dit = generateTestDIT()
	code := m.Run()
	os.Exit(code)
}


func TestGetEntryFindsByDn(t *testing.T) {
	dn := DN{
		[]RDN{
			NewRDN(AVA{"dc", "dev"}),
			NewRDN(AVA{"dc", "georgiboy"}),
			NewRDN(AVA{"cn", "Test1"}),
		}}
	
	if _, err := test_dit.GetEntry(dn); err != nil {
		t.Errorf("Did not retrieve entry from dit: %s", err)
	}
}

func TestGetEntryFailsReturnsMatchedDn(t *testing.T) {
	dn := DN{
		[]RDN{
			NewRDN(AVA{"dc", "dev"}),
			NewRDN(AVA{"dc", "georgiboy"}),
			NewRDN(AVA{"cn", "Nonexistent"}),
		}}

	expectedMatchedDn := DN{
		[]RDN{
			NewRDN(AVA{"dc", "dev"}),
			NewRDN(AVA{"dc", "georgiboy"}),
		}}

	_, err := test_dit.GetEntry(dn)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var nfErr *NodeNotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatalf("Expected not found erro, got %s", err)
	}

	if !reflect.DeepEqual(dn, nfErr.requestedDN) {
		t.Errorf("DN (%s) and requested DN (%s) do not match", dn, nfErr.requestedDN)
	}

	if !reflect.DeepEqual(expectedMatchedDn, nfErr.matchedDn) {
		t.Errorf("Expected matched DN (%s) and nfErr matchedDN (%s) do not match", expectedMatchedDn, nfErr.matchedDn)
	}
}
