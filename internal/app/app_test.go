package app

import (
	"errors"
	"log"
	"path/filepath"
	"runtime"
	"testing"

	d "github.com/georgib0y/relientldap/internal/domain"
	"github.com/georgib0y/relientldap/internal/ldif"
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
	schema, err := ldif.LoadSchmeaFromPaths(attrLdif, ocsLdif)
	if err != nil {
		t.Fatal(err)
	}

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
	}

	for _, test := range tests {
		res, err := bs.Bind(test.req)
		if err != nil {
			if test.err == nil {
				t.Fatalf("Bind service returned unexpected error: %s", err)
			}

			if errors.Is(err, test.err) {
				t.Fatalf("Bind service returned got error: %q but expected error %q", err, test.err)
			}
		}

		if res == nil && test.entryDn != "" {
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
	schema, err := ldif.LoadSchmeaFromPaths(attrLdif, ocsLdif)
	if err != nil {
		t.Fatal(err)
	}

	dit := d.GenerateTestDIT(schema)
	scheduler := NewScheduler(dit, schema)
	defer scheduler.Close()

	as := NewAddService(schema, scheduler)

	tests := []struct {
		req     AddRequest
		addedDn string
		err     error
	}{
		{
			req: TestAddRequest{
				dn: "cn=NewEntry,dc=georgiboy,dc=dev",
				attrs: map[string][]string{
					
				},
			},
		}
	}
}
