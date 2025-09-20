package app

import (
	"errors"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	d "github.com/georgib0y/relientldap/internal/domain"
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

var schema = util.Unwrap(d.LoadSchemaFromReaders(attrLdifFile(), ocsLdifFile()))

type TestSimpleBindRequest struct {
	dn, simple string
}

func (r TestSimpleBindRequest) Dn() string {
	return r.dn
}

func (r TestSimpleBindRequest) Version() int {
	return 3
}

func (r TestSimpleBindRequest) Simple() (string, bool) {
	return r.simple, true
}

func (r TestSimpleBindRequest) SaslMechanism() (string, bool) {
	return "", false
}

func (r TestSimpleBindRequest) SaslCredentials() (string, bool) {
	return "", false
}

func TestBindService(t *testing.T) {
	dit := d.GenerateTestDIT(schema)
	scheduler := NewScheduler(dit, schema)
	defer scheduler.Close()

	bs := NewBindService(schema, scheduler)

	tests := []struct {
		req     BindRequest
		entryDn string
		err     error
	}{
		{
			req:     TestSimpleBindRequest{dn: "cn=Test1,dc=georgiboy,dc=dev", simple: "password123"},
			entryDn: "cn=Test1,dc=georgiboy,dc=dev",
			err:     nil,
		},
		{
			req:     TestSimpleBindRequest{dn: "cn=Test1,dc=georgiboy,dc=dev", simple: "wrong password"},
			entryDn: "cn=Test1,dc=georgiboy,dc=dev",
			err:     d.NewLdapError(d.InvalidCredentials, nil, ""),
		},
	}

	for _, test := range tests {
		res, err := bs.Bind(test.req)
		if err != nil {
			if test.err == nil {
				t.Fatalf("Bind service returned unexpected error: %s", err)
			}

			if !errors.Is(err, test.err) {
				t.Fatalf("Bind service returned error: %q but expected: %q", err, test.err)
			}

			continue
		}

		if res == nil {
			t.Fatalf("Bind service returned nil entry but expected %q", test.entryDn)
		}

		normDn, err := d.NormaliseDN(schema, test.entryDn)
		if err != nil {
			t.Fatal(err)
		}
		testEntry, err := dit.GetEntry(normDn)
		if err != nil {
			t.Fatal(err)
		}

		if testEntry != res {
			t.Fatalf("Bind service returned entry %s but expected %s", res.Dn(), testEntry.Dn())
		}
	}
}

type TestAddRequest struct {
	dn    string
	attrs map[string][]string
}

func (a TestAddRequest) Dn() string {
	return a.dn
}

func (a TestAddRequest) Attributes() map[string][]string {
	return a.attrs
}

func TestAddService(t *testing.T) {
	dit := d.GenerateTestDIT(schema)
	scheduler := NewScheduler(dit, schema)
	defer scheduler.Close()

	as := NewAddService(schema, scheduler)

	tests := []struct {
		req AddRequest
		err error
	}{
		{
			req: TestAddRequest{
				dn: "cn=New Entry,dc=georgiboy,dc=dev",
				attrs: map[string][]string{
					"objectClass": []string{"person"},
					"cn":          []string{"New Entry"},
					"sn":          []string{"Entry"},
				},
			},
			err: nil,
		},
	}

	for _, test := range tests {
		res, err := as.AddEntry(test.req)
		if err != nil {
			if test.err == nil {
				t.Fatalf("Add service returned unexpected err: %s", err)
			}

			if !errors.Is(err, test.err) {
				t.Fatalf("Add service returned error: %q but expected: %q", err, test.err)
			}

			continue
		}

		if res == nil {
			t.Fatalf("Add service returned nil entry, expected %q", test.req.Dn())
		}

		// check that res was put in the expected place
		normDn, err := d.NormaliseDN(schema, test.req.Dn())
		if err != nil {
			t.Fatal(err)
		}
		newEntry, err := dit.GetEntry(normDn)
		if err != nil {
			t.Fatal(err)
		}

		if res != newEntry {
			t.Fatalf("expected res (%p) and newEntry (%p) to be the same pointer", res, newEntry)
		}

		//  test the entry has the expected object classes
		ocs, ok := test.req.Attributes()["objectClass"]
		if !ok {
			t.Fatal("no objectclasses present in the add request")
		}
		for _, name := range ocs {
			oc, ok := schema.FindObjectClass(name)
			if !ok {
				t.Fatalf("unknown object class %q", name)
			}
			if !res.ConatinsObjectClass(oc) {
				t.Fatalf("entry is missing object class %q", name)
			}
		}
		// test the entry has the expected attrs
		for name, vals := range test.req.Attributes() {
			if name == "objectClass" {
				continue
			}
			attr, ok := schema.FindAttribute(name)
			if !ok {
				t.Fatalf("unknown attribute %q", name)
			}
			for _, v := range vals {
				ok, err := res.ContainsAttrVal(attr, v)
				if err != nil {
					t.Fatal(err)
				}
				if !ok {
					t.Fatalf("added entry does not contain value %q", v)
				}
			}

		}

	}
}
