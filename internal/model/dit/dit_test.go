package dit

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

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
	cnTest1 := NewDITNode(dcGeorgiboy, NewEntry(
		WithEntryAttr(AVA{"cn", "Test1"}),
		WithEntryAttr(AVA{"sn", "One"}),
		WithEntryAttr(AVA{"sn", "One-Two"}),
	))
	cnTest2 := NewDITNode(ouTestOu, NewEntry(WithEntryAttr(AVA{"cn", "Test2"})))

	ouTestOu.children[cnTest2] = true
	dcGeorgiboy.children[cnTest1] = true
	dcGeorgiboy.children[ouTestOu] = true
	dcDev.children[dcGeorgiboy] = true

	return DIT{dcDev}
}

func TestMain(m *testing.M) {
	code := m.Run()
	os.Exit(code)
}

func TestGetEntryFindsByDn(t *testing.T) {
	dit := generateTestDIT()
	dn := NewDN(
		WithRdnAva("cn", "Test1"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	if _, err := dit.GetEntry(dn); err != nil {
		t.Errorf("Did not retrieve entry from dit: %s", err)
	}
}

func TestGetEntryFailsReturnsMatchedDn(t *testing.T) {
	dit := generateTestDIT()
	dn := NewDN(
		WithRdnAva("cn", "Nonexistent"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	expectedMatchedDn := NewDN(
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	_, err := dit.GetEntry(dn)

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
	dit := generateTestDIT()
	dn := NewDN(
		WithRdnAva("cn", "New Object"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	entry := NewEntry(
		WithEntryAttr(AVA{"givenName", "New"}),
		WithEntryAttr(AVA{"sn", "Object"}),
	)

	err := dit.InsertEntry(dn, entry)
	if err != nil {
		t.Fatalf("Error inserting new entry: %s", err)
	}

	entry, err = dit.GetEntry(dn)
	if err != nil {
		t.Fatalf("Error retrieving new entry after inserting: %s", err)
	}

	expAttrs := []AVA{
		AVA{"givenName", "New"},
		AVA{"sn", "Object"},
		AVA{"cn", "New Object"},
	}

	for _, ava := range expAttrs {
		if !entry.ContainsAttr(ava) {
			t.Errorf("Entry is missing attr: %s", ava)
		}
	}

}

func TestDeleteEntryDeletesNode(t *testing.T) {
	dit := generateTestDIT()
	dn := NewDN(
		WithRdnAva("cn", "Test1"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	if err := dit.DeleteEntry(dn); err != nil {
		t.Fatal("Error deleting entry: ", err)
	}

	_, err := dit.GetEntry(dn)

	var nfErr *NodeNotFoundError
	if !errors.As(err, &nfErr) {
		t.Fatal("Unexpected error getting deleted entry: ", err)
	} else if err == nil {
		t.Fatal("Expected not found error when getting deleted entry, got nil")
	}
}

func TestDeleteEntryFailsOnNonLeafNode(t *testing.T) {
	dit := generateTestDIT()
	dn := NewDN(
		WithRdnAva("ou", "TestOu"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	err := dit.DeleteEntry(dn)

	if !errors.Is(err, ErrNodeNotLeaf) {
		t.Fatal("Expected Node Not Leaf error, got: ", err)
	}
}

func TestModifyAddEntryAddsAttributes(t *testing.T) {
	dit := generateTestDIT()
	dn := NewDN(
		WithRdnAva("cn", "Test1"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	if err := dit.ModifyEntry(dn, AddOperation("fax", "12345")); err != nil {
		t.Fatal("Got error when adding entry: ", err)
	}

	ok, err := dit.ContainsAttribute(dn, AVA{"fax", "12345"})
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if !ok {
		t.Fatal("Attribute not added to entry: ", dn)
	}
}

func TestModifyDeleteSingleEntryDeletesAttribute(t *testing.T) {
	dit := generateTestDIT()
	dn := NewDN(
		WithRdnAva("cn", "Test1"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	if err := dit.ModifyEntry(dn, DeleteOperation("sn", "One-Two")); err != nil {
		t.Fatal("Got error when deleting entry: ", err)
	}

	ok, err := dit.ContainsAttribute(dn, AVA{"sn", "One-Two"})
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if ok {
		t.Fatal("Attribute not deleted to entry: ", dn)
	}

	ok, err = dit.ContainsAttribute(dn, AVA{"sn", "One"})
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if !ok {
		t.Fatal("Attribute deleted from entry: ", dn)
	}
}

func TestModifyDeleteAllEntryDeletesAttributes(t *testing.T) {
	dit := generateTestDIT()
	dn := NewDN(
		WithRdnAva("cn", "Test1"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	if err := dit.ModifyEntry(dn, DeleteOperation("sn")); err != nil {
		t.Fatal("Got error when deleting entry: ", err)
	}

	ok, err := dit.ContainsAttribute(dn, AVA{"sn", "One-Two"})
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if ok {
		t.Fatal("Attribute not deleted to entry: ", dn)
	}

	ok, err = dit.ContainsAttribute(dn, AVA{"sn", "One"})
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if ok {
		t.Fatal("Attribute not deleted to entry: ", dn)
	}
}

func TestModifyReplaceReplacesAttributes(t *testing.T) {
	dit := generateTestDIT()
	dn := NewDN(
		WithRdnAva("cn", "Test1"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	if err := dit.ModifyEntry(dn, ReplaceOperation("sn", "Three", "Three-Four")); err != nil {
		t.Fatal("Got error when replacing entry: ", err)
	}

	ok, err := dit.ContainsAttribute(dn, AVA{"sn", "One-Two"})
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if ok {
		t.Fatal("Attribute not deleted from entry: ", dn)
	}

	ok, err = dit.ContainsAttribute(dn, AVA{"sn", "One"})
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if ok {
		t.Fatal("Attribute not deleted from entry: ", dn)
	}

	ok, err = dit.ContainsAttribute(dn, AVA{"sn", "Three"})
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if !ok {
		t.Fatal("Attribute Three not added to entry: ", dn)
	}

	ok, err = dit.ContainsAttribute(dn, AVA{"sn", "Three-Four"})
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if !ok {
		t.Fatal("Attribute Three-Four not added to entry: ", dn)
	}
}

func TestModifyDNChangesRDNAndMovesEntry(t *testing.T) {
	dit := generateTestDIT()

	dn := NewDN(
		WithRdnAva("cn", "Test1"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	rdn := NewRDN(WithAVA("givenName", "Test1Moved"))

	newSuperDn := NewDN(
		WithRdnAva("ou", "TestOu"),
		WithRdnAva("dc", "georgiboy"),
		WithRdnAva("dc", "dev"),
	)

	err := dit.ModifyEntryDN(dn, rdn, true, &newSuperDn)
	if err != nil {
		t.Fatal("Failed to modify dn: ", err)
	}

	newSuperDn.AddRDN(rdn)
	newDn := newSuperDn

	entry, err := dit.GetEntry(newDn)
	if err != nil {
		t.Fatal("Failed to refetch entry after moving: ", err)
	}

	if !CompareDNs(entry.dn, newDn) {
		t.Fatalf("Failed to update DN of entry, got: %s expected: %s", entry.dn, newDn)
	}
	entry.ContainsAttr(AVA{"givenName", "Test1Moved"})
}
