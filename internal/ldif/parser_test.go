package ldif

import (
	"strings"
	"testing"

	"github.com/georgib0y/relientldap/internal/model/schema"
)

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


`

func TestParsesObjectClass(t *testing.T) {
	tests := []struct {
		ldif string
		exp  []schema.ObjectClass
	}{
		{
			ldif: `( 2.5.6.0 NAME 'top' DESC 'this is some desc' )`,
			exp: []schema.ObjectClass{schema.NewObjectClass(
				schema.ObjClassWithOid("2.5.6.0"),
				schema.ObjClassWithName("top"),
				schema.ObjClassWithDesc("this is some desc"),
			)},
		},
		{
			ldif: `( 2.5.6.0 NAME 'top' DESC 'somethign' )
( 2.5.6.1 NAME 'secondTop')`,
			exp: []schema.ObjectClass{
				schema.NewObjectClass(
					schema.ObjClassWithOid("2.5.6.0"),
					schema.ObjClassWithName("top"),
					schema.ObjClassWithDesc("somethign"),
				),
				schema.NewObjectClass(
					schema.ObjClassWithOid("2.5.6.1"),
					schema.ObjClassWithName("secondTop"),
				),
			},
		},
		{
			ldif: `( 2.5.6.6 NAME ( 'person' 'ps' ) )`,
			exp: []schema.ObjectClass{schema.NewObjectClass(
				schema.ObjClassWithOid("2.5.6.6"),
				schema.ObjClassWithName("person", "ps"),
			)},
		},
		{
			ldif: `( 2.5.6.6 NAME ( 'person' 'ps' ) )`,
			exp: []schema.ObjectClass{schema.NewObjectClass(
				schema.ObjClassWithOid("2.5.6.6"),
				schema.ObjClassWithName("person", "ps"),
			)},
		},
		{
			ldif: manyOcDefs,
			exp: []schema.ObjectClass{
				schema.NewObjectClass(
					schema.ObjClassWithOid("2.5.6.11"),
					schema.ObjClassWithName("applicationProcess"),
					schema.ObjClassWithSupOid("top"),
					schema.ObjClassWithKind(schema.Structural),
					schema.ObjClassWithMustAttr("cn"),
					schema.ObjClassWithMayAttr("seeAlso", "ou", "l", "description"),
				),
				schema.NewObjectClass(
					schema.ObjClassWithOid("2.5.6.2"),
					schema.ObjClassWithName("country"),
					schema.ObjClassWithSupOid("top"),
					schema.ObjClassWithKind(schema.Structural),
					schema.ObjClassWithMustAttr("c"),
					schema.ObjClassWithMayAttr("searchGuide", "description"),
				),
				schema.NewObjectClass(
					schema.ObjClassWithOid("1.3.6.1.4.1.1466.344"),
					schema.ObjClassWithName("dcObject"),
					schema.ObjClassWithSupOid("top"),
					schema.ObjClassWithKind(schema.Auxilary),
					schema.ObjClassWithMustAttr("dc"),
				),
				schema.NewObjectClass(
					schema.ObjClassWithOid("2.5.6.14"),
					schema.ObjClassWithName("device"),
					schema.ObjClassWithSupOid("top"),
					schema.ObjClassWithKind(schema.Structural),
					schema.ObjClassWithMustAttr("cn"),
					schema.ObjClassWithMayAttr("serialNumber", "seeAlso", "owner", "ou", "o", "l", "description"),
				),
				schema.NewObjectClass(
					schema.ObjClassWithOid("2.5.6.9"),
					schema.ObjClassWithName("groupOfNames"),
					schema.ObjClassWithSupOid("top"),
					schema.ObjClassWithKind(schema.Structural),
					schema.ObjClassWithMustAttr("member", "cn"),
					schema.ObjClassWithMayAttr("businessCategory", "seeAlso", "owner", "ou", "o", "description"),
				),
			},
		},
	}

	p := NewObjectClassParser()
	for _, test := range tests {
		oc, err := ParseReader(strings.NewReader(test.ldif), p)
		if err != nil {
			t.Fatalf("Failed to parse ldif:\n%s\nErr is: %s", test.ldif, err)
		}

		if len(oc) != len(test.exp) {
			t.Fatalf("parsed wrong number of object classes, got %d expected %d", len(oc), len(test.exp))
		}

		for i := range test.exp {
			if err = schema.ObjectClassesAreEqual(oc[i], test.exp[i]); err != nil {
				t.Fatalf("Parsed object class did not match expected for ldif:\n%s\nGot: %s\nExp: %s\nReason: %s", test.ldif, oc[i], test.exp[i], err)
			}
		}

		t.Logf("passed parsed ldif: %s", test.ldif)
	}
}
