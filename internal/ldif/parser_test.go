package ldif

import (
	"strings"
	"testing"

	"github.com/georgib0y/relientldap/internal/model/schema"
)

var attributes = map[schema.OID]*schema.Attribute{
	"cn":               schema.NewAttributeBuilder().SetOid("cn").Build(),
	"c":                schema.NewAttributeBuilder().SetOid("c").Build(),
	"seeAlso":          schema.NewAttributeBuilder().SetOid("seeAlso").Build(),
	"ou":               schema.NewAttributeBuilder().SetOid("ou").Build(),
	"l":                schema.NewAttributeBuilder().SetOid("l").Build(),
	"description":      schema.NewAttributeBuilder().SetOid("description").Build(),
	"searchGuide":      schema.NewAttributeBuilder().SetOid("searchGuide").Build(),
	"dc":               schema.NewAttributeBuilder().SetOid("dc").Build(),
	"serialNumber":     schema.NewAttributeBuilder().SetOid("serialNumber").Build(),
	"owner":            schema.NewAttributeBuilder().SetOid("owner").Build(),
	"o":                schema.NewAttributeBuilder().SetOid("o").Build(),
	"businessCategory": schema.NewAttributeBuilder().SetOid("businessCategory").Build(),
	"member":           schema.NewAttributeBuilder().SetOid("member").Build(),
}

func getAttrs(oid ...string) []*schema.Attribute {
	attrs := []*schema.Attribute{}
	for _, o := range oid {
		attrs = append(attrs, attributes[schema.OID(o)])
	}
	return attrs
}

var objectClasses = map[schema.OID]*schema.ObjectClass{
	"top": schema.NewObjectClassBuilder().SetOid("2.5.6.0").Build(),
}

func getObjClass(oid ...string) []*schema.ObjectClass {
	ocs := []*schema.ObjectClass{}
	for _, o := range oid {
		ocs = append(ocs, objectClasses[schema.OID(o)])
	}
	return ocs
}

func TestTokeniserTokenisesTokens(t *testing.T) {
	tests := []struct {
		ldif string
		exp  []Token
	}{
		{
			ldif: "( 0.1.2.3 NAME 'name' DESC 'some description' )",
			exp: []Token{
				Token{"(", LPAREN},
				Token{"0.1.2.3", NUMERICOID},
				Token{"NAME", KEYWORD},
				Token{"'name'", QDESCR},
				Token{"DESC", KEYWORD},
				Token{"'some description'", QDSTRING},
				Token{")", RPAREN},
			},
		},
	}

	for _, test := range tests {
		r := strings.NewReader(test.ldif)
		tokens, err := tokenise(r)
		if err != nil {
			t.Fatalf("failed to parse:\n\t%s\n\nerr is: %s", test.ldif, err)
		}

		if len(tokens) != len(test.exp) {
			t.Fatalf("wrong number of tokens! got %d expected %d", len(tokens), len(test.exp))
		}

		for i := range tokens {
			if tokens[i].val != test.exp[i].val {
				t.Fatalf("wrong token value, got %s expected %s", tokens[i].val, test.exp[i].val)
			}
			if tokens[i].tokenType != test.exp[i].tokenType {
				t.Fatalf("wrong token value, got %s expected %s", tokens[i].tokenType, test.exp[i].tokenType)
			}
		}
	}
}

const manyOcDefs = `
	( 2.5.6.11 NAME 'applicationProcess'
         SUP top
         STRUCTURAL
         MUST cn
         MAY ( seeAlso $
               ou $
               l $
               description ) )


      ( 2.5.6.2 NAME 'country'
         SUP top
         STRUCTURAL
         MUST c
         MAY ( searchGuide $
               description ) )

      ( 1.3.6.1.4.1.1466.344 NAME 'dcObject'
         SUP top
         AUXILIARY
         MUST dc )

      ( 2.5.6.14 NAME 'device'
         SUP top
         STRUCTURAL
         MUST cn
         MAY ( serialNumber $
               seeAlso $
               owner $
               ou $
               o $
               l $
               description ) )

      ( 2.5.6.9 NAME 'groupOfNames'
         SUP top
         STRUCTURAL
         MUST ( member $
               cn )
         MAY ( businessCategory $
               seeAlso $
               owner $
               ou $
               o $
               description ) )

	( 2.5.6.0 NAME 'top' DESC 'this is some desc' )
`

func TestParsesObjectClass(t *testing.T) {
	tests := []struct {
		ldif string
		exp  []*schema.ObjectClass
	}{
		{
			ldif: `( 2.5.6.0 NAME 'top' DESC 'this is some desc' )`,
			exp: []*schema.ObjectClass{schema.NewObjectClassBuilder().
				SetOid("2.5.6.0").
				AddName("top").
				SetDesc("this is some desc").
				Build(),
			},
		},
		{
			ldif: `( 2.5.6.0 NAME 'top' DESC 'somethign' )
( 2.5.6.1 NAME 'secondTop')`,
			exp: []*schema.ObjectClass{
				schema.NewObjectClassBuilder().SetOid("2.5.6.0").AddName("top").SetDesc("somethign").Build(),
				schema.NewObjectClassBuilder().SetOid("2.5.6.1").AddName("secondTop").Build(),
			},
		},
		{
			ldif: `( 2.5.6.6 NAME ( 'person' 'ps' ) )`,
			exp: []*schema.ObjectClass{
				schema.NewObjectClassBuilder().SetOid("2.5.6.6").AddName("person", "ps").Build(),
			},
		},
		{
			ldif: `( 2.5.6.6 NAME ( 'person' 'ps' ) )`,
			exp: []*schema.ObjectClass{
				schema.NewObjectClassBuilder().SetOid("2.5.6.6").AddName("person", "ps").Build(),
			},
		},
		{
			ldif: manyOcDefs,
			exp: []*schema.ObjectClass{
				schema.NewObjectClassBuilder().
					SetOid("2.5.6.11").
					AddName("applicationProcess").
					AddSup(getObjClass("top")...).
					SetKind(schema.Structural).
					AddMustAttr(getAttrs("cn")...).
					AddMayAttr(getAttrs("seeAlso", "ou", "l", "description")...).
					Build(),
				schema.NewObjectClassBuilder().
					SetOid("2.5.6.2").
					AddName("country").
					AddSup(getObjClass("top")...).
					SetKind(schema.Structural).
					AddMustAttr(getAttrs("c")...).
					AddMayAttr(getAttrs("searchGuide", "description")...).
					Build(),
				schema.NewObjectClassBuilder().
					SetOid("1.3.6.1.4.1.1466.344").
					AddName("dcObject").
					AddSup(getObjClass("top")...).
					SetKind(schema.Auxilary).
					AddMustAttr(getAttrs("dc")...).
					Build(),
				schema.NewObjectClassBuilder().
					SetOid("2.5.6.14").
					AddName("device").
					AddSup(getObjClass("top")...).
					SetKind(schema.Structural).
					AddMustAttr(getAttrs("cn")...).
					AddMayAttr(getAttrs("serialNumber", "seeAlso", "owner", "ou", "o", "l", "description")...).
					Build(),
				schema.NewObjectClassBuilder().
					SetOid("2.5.6.9").
					AddName("groupOfNames").
					AddSup(getObjClass("top")...).
					SetKind(schema.Structural).
					AddMustAttr(getAttrs("member", "cn")...).
					AddMayAttr(getAttrs("businessCategory", "seeAlso", "owner", "ou", "o", "description")...).
					Build(),
				schema.NewObjectClassBuilder().
					SetOid("2.5.6.0").
					AddName("top").
					SetDesc("this is some desc").
					Build(),
			},
		},
	}

	p := NewObjectClassParser(attributes)
	for _, test := range tests {
		parsed_objectClasses, err := ParseReader(strings.NewReader(test.ldif), p)
		if err != nil {
			t.Fatalf("Failed to parse ldif:\n%s\nErr is: %s", test.ldif, err)
		}

		if len(parsed_objectClasses) != len(test.exp) {
			t.Fatalf("parsed wrong number of object classes, got %d expected %d", len(parsed_objectClasses), len(test.exp))
		}

		for _, e := range test.exp {
			objClass, ok := parsed_objectClasses[e.Oid()]
			if !ok {
				t.Fatalf("Parsed object classes does not contain %s", e.Oid())
			}

			if err = schema.ObjectClassesAreEqual(objClass, e); err != nil {
				t.Fatalf("Parsed object class did not match expected for ldif:\n%s\nGot: %s\nExp: %s\nReason: %s", test.ldif, objClass, e, err)
			}
		}

		t.Logf("passed parsed ldif: %s", test.ldif)
	}
}
