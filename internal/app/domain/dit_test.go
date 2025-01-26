package domain

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

var testDit DIT

/*
Test DIT Structue
dc=dev
dc=georgiboy
cn=Test1 | ou=TestOu
         | cn=Test2
*/

func generateTestDIT() DIT {
	dcDev := NewDITNode(nil, NewEntry(WithEntryAttr(AVA{"dc", "dev"})))
	dcGeorgiboy := NewDITNode(dcDev, NewEntry(WithEntryAttr(AVA{"dc", "georgiboy"})))
	ouTestOu := NewDITNode(dcGeorgiboy, NewEntry(WithEntryAttr(AVA{"ou", "TestOu"})))
	cnTest1 := NewDITNode(dcGeorgiboy, NewEntry(WithEntryAttr(AVA{"cn", "Test1"})))
	cnTest2 := NewDITNode(ouTestOu, NewEntry(WithEntryAttr(AVA{"cn", "Test2"})))

	ouTestOu.children[cnTest2] = true
	dcGeorgiboy.children[cnTest1] = true
	dcGeorgiboy.children[ouTestOu] = true
	dcDev.children[dcGeorgiboy] = true

	return DIT{dcDev}
}

func TestMain(m *testing.M) {
	testDit = generateTestDIT()
	code := m.Run()
	os.Exit(code)
}

func TestGetEntryFindsByDn(t *testing.T) {
	dn := NewDN(
		WithRdnAva("cn", "Test1"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	if _, err := testDit.GetEntry(dn); err != nil {
		t.Errorf("Did not retrieve entry from dit: %s", err)
	}
}

func TestGetEntryFailsReturnsMatchedDn(t *testing.T) {
	dn := NewDN(
		WithRdnAva("cn", "Nonexistent"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	expectedMatchedDn := NewDN(
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	_, err := testDit.GetEntry(dn)

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

func TestInsertEntryPutsEntryInTreeWithRdnAtt(t *testing.T) {
	dn := NewDN(
		WithRdnAva("cn", "New Object"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	entry := NewEntry(
		WithEntryAttr(AVA{"givenName", "New"}),
		WithEntryAttr(AVA{"sn", "Object"}),
	)

	err := testDit.InsertEntry(dn, entry)
	if err != nil {
		t.Fatalf("Error inserting new entry: %s", err)
	}

	entry, err = testDit.GetEntry(dn)
	if err != nil {
		t.Fatalf("Error retrieving new entry after inserting: %s", err)
	}

	expAttrs := map[OID]string{
		"givenName": "New",
		"sn":        "Object",
		"cn":        "New Object",
	}

	for o, v := range expAttrs {
		if !entry.ContainsAttr(o, v) {
			t.Errorf("Entry is missing attr: %s: %s", o, v)
		}
	}

}

func TestDeleteEntryDeletesNode(t *testing.T) {
	dn := NewDN(
		WithRdnAva("cn", "Test1"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	if err := testDit.DeleteEntry(dn); err != nil {
		t.Fatal("Error deleting entry: ", err)
	}

	_, err := testDit.GetEntry(dn)

	var nfErr *NodeNotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatal("Unexpected error getting deleted entry: ", err)
	} else if err == nil {
		t.Fatal("Expected not found error when getting deleted entry, got nil")
	}
}

func TestDeleteEntryFailsOnNonLeafNode(t *testing.T) {
	dn := NewDN(
		WithRdnAva("ou", "TestOu"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	err := testDit.DeleteEntry(dn)

	if !errors.Is(err, ErrNodeNotLeaf) {
		t.Fatal("Expected Node Not Leaf error, got: ", err)
	}
}

func TestModifyAddEntryAddsAttributes(t *testing.T) {
	dn := NewDN(
		WithRdnAva("cn", "Test1"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	testDit.ModifyEntry
}
