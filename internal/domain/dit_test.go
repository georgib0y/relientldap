package domain

import (
	"errors"
	"os"

	// "github.com/georgib0y/relientldap/internal/ldif"
	"log"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/georgib0y/relientldap/internal/util"
)

var (
	rootDir  = projectRootDir()
	attrLdif = filepath.Join(rootDir, "ldif/attributes.ldif")
	ocsLdif  = filepath.Join(rootDir, "ldif/objClasses.ldif")
)

func projectRootDir() string {
	_, f, _, ok := runtime.Caller(0)
	if !ok {
		log.Panic("runtime.Caller(0) not ok")
	}

	root := filepath.Join(filepath.Dir(f), "../..")
	log.Print(root)
	return root
}

// opens attr ldif or panics
func attrLdifFile() *os.File {
	f, err := os.Open(attrLdif)
	if err != nil {
		log.Panicf("couldnt open attr ldif file: %s", attrLdif)
	}
	return f
}

// opens attr ldif or panics
func ocsLdifFile() *os.File {
	f, err := os.Open(ocsLdif)
	if err != nil {
		log.Panicf("couldnt open object class ldif file: %s", ocsLdif)
	}
	return f
}

// const attrLdif string = `
//       ( 0.9.2342.19200300.100.1.25 NAME 'dc'
//          EQUALITY caseIgnoreIA5Match
//          SUBSTR caseIgnoreIA5SubstringsMatch
//          SYNTAX 1.3.6.1.4.1.1466.115.121.1.26
//          SINGLE-VALUE )

//       ( 2.5.4.41 NAME 'name'
//          EQUALITY caseIgnoreMatch
//          SUBSTR caseIgnoreSubstringsMatch
//          SYNTAX 1.3.6.1.4.1.1466.115.121.1.15 )

//       ( 2.5.4.11 NAME 'ou'
//          SUP name )

//       ( 2.5.4.3 NAME 'cn'
//          SUP name )

//       ( 2.5.4.4 NAME 'sn'
//          SUP name )

// 	  ( 2.5.4.23 NAME 'facsimileTelephoneNumber'
//          SYNTAX 1.3.6.1.4.1.1466.115.121.1.22 )

//       ( 2.5.4.42 NAME 'givenName'
//          SUP name )
// `

// var attrs = map[OID]*Attribute{
// 	"dc":                       NewAttributeBuilder().SetOid("dc").AddNames("dc").Build(),
// 	"ou":                       NewAttributeBuilder().SetOid("ou").AddNames("ou").Build(),
// 	"cn":                       NewAttributeBuilder().SetOid("cn").AddNames("cn").Build(),
// 	"sn":                       NewAttributeBuilder().SetOid("sn").AddNames("sn").Build(),
// 	"facsimileTelephoneNumber": NewAttributeBuilder().SetOid("facsimileTelephoneNumber").Build(),
// 	"givenName":                NewAttributeBuilder().SetOid("givenName").Build(),
// }

var schema = util.Unwrap(LoadSchemaFromReaders(attrLdifFile(), ocsLdifFile()))
var attrs = map[string]*Attribute{
	"dc":                       util.UnwrapOk(schema.FindAttribute("dc")),
	"ou":                       util.UnwrapOk(schema.FindAttribute("ou")),
	"cn":                       util.UnwrapOk(schema.FindAttribute("cn")),
	"sn":                       util.UnwrapOk(schema.FindAttribute("sn")),
	"facsimileTelephoneNumber": util.UnwrapOk(schema.FindAttribute("facsimileTelephoneNumber")),
	"givenName":                util.UnwrapOk(schema.FindAttribute("givenName")),
}

var objClasses = map[string]*ObjectClass{
	"person": util.UnwrapOk(schema.FindObjectClass("person")),
}

func TestGetEntryFindsByDn(t *testing.T) {
	dit := GenerateTestDIT(schema)
	dn := NewDnBuilder().AddNamingContext(attrs["dc"], "dev", "georgiboy").AddAvaAsRdn(attrs["cn"], "Test1").Build()

	if _, err := dit.GetEntry(dn); err != nil {
		t.Errorf("Did not retrieve entry from dit: %s", err)
	}
}

func TestGetEntryFailsReturnsMatchedDn(t *testing.T) {
	dit := GenerateTestDIT(schema)
	dn := NewDnBuilder().AddNamingContext(attrs["dc"], "dev", "georgiboy").AddAvaAsRdn(attrs["cn"], "Nonexistent").Build()

	expectedMatchedDn := NewDnBuilder().AddNamingContext(attrs["dc"], "dev", "georgiboy").Build()

	_, err := dit.GetEntry(dn)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var ldapErr LdapError
	if !errors.As(err, &ldapErr) {
		t.Fatalf("Expected ldap err, got %s", err)
	}

	if ldapErr.ResultCode != NoSuchObject {
		t.Fatalf("Expected NoSuchObject ldap err, got %s", ldapErr.ResultCode)
	}

	if ldapErr.MatchedDN == nil {
		t.Fatal("Expected matched dn got nil")
	}

	if !CompareDNs(expectedMatchedDn, *ldapErr.MatchedDN) {
		t.Errorf("Expected matched DN (%s) and nfErr matchedDN (%s) do not match", expectedMatchedDn, ldapErr.MatchedDN)
	}
}

func TestInsertEntryPutsEntryInTreeWithRdnAtt(t *testing.T) {
	dit := GenerateTestDIT(schema)
	dn := NewDnBuilder().AddNamingContext(attrs["dc"], "dev", "georgiboy").AddAvaAsRdn(attrs["cn"], "New Object").Build()

	entry, err := NewEntry(schema, dn,
		WithStructural(objClasses["person"]),
		WithEntryAttr(attrs["sn"], "Object"),
	)
	if err != nil {
		t.Fatal(err)
	}

	err = dit.InsertEntry(dn, entry)
	if err != nil {
		t.Fatalf("Error inserting new entry: %s", err)
	}

	entry, err = dit.GetEntry(dn)
	if err != nil {
		t.Fatalf("Error retrieving new entry after inserting: %s", err)
	}

	expAttrs := []struct {
		attr *Attribute
		val  string
	}{
		// {attrs["givenName"], "New"},
		{attrs["sn"], "Object"},
		{attrs["cn"], "New Object"},
	}

	for _, exp := range expAttrs {
		contains, err := entry.ContainsAttrVal(exp.attr, exp.val)

		if err != nil {
			t.Errorf("Error matching attr: %s %s, %s", exp.attr.Oid(), exp.val, err)
		}

		if !contains {
			t.Errorf("Entry is missing attr: %s %s", exp.attr.Oid(), exp.val)
		}
	}

}

func TestDeleteEntryDeletesNode(t *testing.T) {
	dit := GenerateTestDIT(schema)
	dn := NewDnBuilder().AddNamingContext(attrs["dc"], "dev", "georgiboy").AddAvaAsRdn(attrs["cn"], "Test1").Build()

	if err := dit.DeleteEntry(dn); err != nil {
		t.Fatal("Error deleting entry: ", err)
	}

	_, err := dit.GetEntry(dn)

	var ldapErr LdapError
	if !errors.As(err, &ldapErr) {
		if ldapErr.ResultCode != NoSuchObject {
			t.Fatal("Expected ldap nosuchobject error getting deleted entry, got: ", err)
		}
	} else if err == nil {
		t.Fatal("Expected ldap nosuchobject error when getting deleted entry, got nil")
	}
}

func TestDeleteEntryFailsOnNonLeafNode(t *testing.T) {
	dit := GenerateTestDIT(schema)
	dn := NewDnBuilder().AddNamingContext(attrs["dc"], "dev", "georgiboy").AddAvaAsRdn(attrs["ou"], "TestOu").Build()

	err := dit.DeleteEntry(dn)

	if !errors.Is(err, ErrNodeNotLeaf) {
		t.Fatal("Expected Node Not Leaf error, got: ", err)
	}
}

func TestModifyAddEntryAddsAttributes(t *testing.T) {
	dit := GenerateTestDIT(schema)
	dn := NewDnBuilder().AddNamingContext(attrs["dc"], "dev", "georgiboy").AddAvaAsRdn(attrs["cn"], "Test1").Build()

	if err := dit.ModifyEntry(dn, AddOperation(attrs["facsimileTelephoneNumber"], "12345")); err != nil {
		t.Fatal("Got error when adding entry: ", err)
	}

	ok, err := dit.ContainsAttribute(dn, attrs["facsimileTelephoneNumber"], "12345")
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if !ok {
		t.Fatal("Attribute not added to entry: ", dn)
	}
}

func TestModifyDeleteSingleEntryDeletesAttribute(t *testing.T) {
	dit := GenerateTestDIT(schema)
	dn := NewDnBuilder().AddNamingContext(attrs["dc"], "dev", "georgiboy").AddAvaAsRdn(attrs["cn"], "Test1").Build()

	if err := dit.ModifyEntry(dn, DeleteOperation(attrs["sn"], "One-Two")); err != nil {
		t.Fatal("Got error when deleting entry: ", err)
	}

	ok, err := dit.ContainsAttribute(dn, attrs["sn"], "One-Two")
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if ok {
		t.Fatal("Attribute not deleted to entry: ", dn)
	}

	ok, err = dit.ContainsAttribute(dn, attrs["sn"], "One")
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if !ok {
		t.Fatal("Attribute deleted from entry: ", dn)
	}
}

func TestModifyDeleteAllEntryDeletesAttributes(t *testing.T) {
	dit := GenerateTestDIT(schema)
	dn := NewDnBuilder().AddNamingContext(attrs["dc"], "dev", "georgiboy").AddAvaAsRdn(attrs["cn"], "Test1").Build()

	if err := dit.ModifyEntry(dn, DeleteOperation(attrs["sn"])); err != nil {
		t.Fatal("Got error when deleting entry: ", err)
	}

	ok, err := dit.ContainsAttribute(dn, attrs["sn"], "One-Two")
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if ok {
		t.Fatal("Attribute not deleted to entry: ", dn)
	}

	ok, err = dit.ContainsAttribute(dn, attrs["sn"], "One")
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if ok {
		t.Fatal("Attribute not deleted to entry: ", dn)
	}
}

func TestModifyReplaceReplacesAttributes(t *testing.T) {
	dit := GenerateTestDIT(schema)
	dn := NewDnBuilder().AddNamingContext(attrs["dc"], "dev", "georgiboy").AddAvaAsRdn(attrs["cn"], "Test1").Build()

	if err := dit.ModifyEntry(dn, ReplaceOperation(attrs["sn"], "Three", "Three-Four")); err != nil {
		t.Fatal("Got error when replacing entry: ", err)
	}

	ok, err := dit.ContainsAttribute(dn, attrs["sn"], "One-Two")
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if ok {
		t.Fatal("Attribute not deleted from entry: ", dn)
	}

	ok, err = dit.ContainsAttribute(dn, attrs["sn"], "One")
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if ok {
		t.Fatal("Attribute not deleted from entry: ", dn)
	}

	ok, err = dit.ContainsAttribute(dn, attrs["sn"], "Three")
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if !ok {
		t.Fatal("Attribute Three not added to entry: ", dn)
	}

	ok, err = dit.ContainsAttribute(dn, attrs["sn"], "Three-Four")
	if err != nil {
		t.Fatal("Error when comparing attr: ", err)
	}

	if !ok {
		t.Fatal("Attribute Three-Four not added to entry: ", dn)
	}
}

/*
Transform this DIT Structue:
| dc=dev
--| dc=georgiboy
----| cn=Test1
----| ou=TestOu
------| cn=Test2

into:
| dc=dev
--| dc=georgiboy
----| ou=TestOu
------| givenName=Test1Moved
------| cn=Test2
*/
func TestModifyDNChangesRDNDeletesOldRDNAndMovesEntry(t *testing.T) {
	dit := GenerateTestDIT(schema)

	dn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		AddAvaAsRdn(attrs["cn"], "Test1").
		Build()

	rdn := NewRDN(WithAVA(attrs["givenName"], "Test1Moved"))

	newSuperDn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		AddAvaAsRdn(attrs["ou"], "TestOu").
		Build()

	var sb1 strings.Builder
	WriteNodeDescendants(&sb1, dit.root)
	t.Log(sb1.String())

	err := dit.ModifyEntryDN(dn, rdn, true, &newSuperDn)
	if err != nil {
		t.Fatal("Failed to modify dn: ", err)
	}

	var sb2 strings.Builder
	WriteNodeDescendants(&sb2, dit.root)
	t.Log(sb2.String())

	newDn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		AddAvaAsRdn(attrs["ou"], "TestOu").
		AddAvaAsRdn(attrs["givenName"], "Test1Moved").
		Build()

	entry, err := dit.GetEntry(newDn)
	if err != nil {
		t.Fatal("Failed to refetch entry after moving: ", err)
	}

	if !CompareDNs(entry.dn, newDn) {
		t.Fatalf("Failed to update DN of entry, got: %s expected: %s", entry.dn, newDn)
	}

	entry.ContainsAttrVal(attrs["givenName"], "Test1Moved")
}

func TestSearchBaseObject(t *testing.T) {
	dit := GenerateTestDIT(schema)

	baseDn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		AddAvaAsRdn(attrs["cn"], "Test1").
		Build()

	entry, err := dit.GetEntry(baseDn)
	if err != nil {
		t.Fatalf("failed to get entry: %s", err)
	}

	matchingFilter := NewEqualityFilter(attrs["cn"], "Test1")
	res, err := dit.Search(baseDn, BaseObject, matchingFilter)
	if err != nil {
		t.Fatalf("error in base object search: %s", err)
	}

	if len(res) != 1 {
		for _, r := range res {
			t.Log(r)
		}
		t.Fatalf("base object search expected 1 entry but got %d", len(res))
	}

	if res[0] != entry {
		t.Fatalf("search returned wrong entry")
	}

	nonMatchingFilter := NewEqualityFilter(attrs["cn"], "unknown")
	res, err = dit.Search(baseDn, BaseObject, nonMatchingFilter)
	if err != nil {
		t.Fatalf("error in base object search: %s", err)
	}
	if len(res) > 0 {
		t.Fatalf("expected no results, got %d", len(res))
	}
}

func TestSearchSingleLevel(t *testing.T) {
	dit := GenerateTestDIT(schema)

	baseDn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		AddAvaAsRdn(attrs["ou"], "TestOu").
		Build()

	filter := NewEqualityFilter(attrs["cn"], "Test2")
	res, err := dit.Search(baseDn, SingleLevel, filter)
	if err != nil {
		t.Fatalf("error in single level search: %s", err)
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}

	ok, err := res[0].MatchesRdn(NewRDN(WithAVA(attrs["cn"], "Test2")))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Log(res[0])
		t.Fatal("expected {cn:Test2} to be rdn for result")
	}

	filter = NewEqualityFilter(attrs["sn"], "Tester")
	res, err = dit.Search(baseDn, SingleLevel, filter)
	if err != nil {
		t.Fatalf("error in single level search: %s", err)
	}

	if len(res) != 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
}

func TestSearchWholeSubtree(t *testing.T) {
	dit := GenerateTestDIT(schema)

	baseDn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		Build()

	filter := NewEqualityFilter(attrs["cn"], "Test2")
	res, err := dit.Search(baseDn, WholeSubtree, filter)
	if err != nil {
		t.Fatalf("error in whole subtree search: %s", err)
	}

	if len(res) != 1 {
		t.Fatalf("expected 1 result, got %d", len(res))
	}

	ok, err := res[0].MatchesRdn(NewRDN(WithAVA(attrs["cn"], "Test2")))
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Log(res[0])
		t.Fatal("expected {cn:Test2} to be rdn for only result")
	}

	filter = NewEqualityFilter(attrs["sn"], "Tester")
	res, err = dit.Search(baseDn, WholeSubtree, filter)
	if err != nil {
		t.Fatalf("error in whole subtree search: %s", err)
	}

	if len(res) != 3 {
		t.Fatalf("expected 3 results, got %d", len(res))
	}
}
