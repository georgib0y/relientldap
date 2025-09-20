package domain

import (
	"strings"
	"testing"

	"github.com/georgib0y/relientldap/internal/util"
)

var attributes = map[OID]*Attribute{
	"cn":               NewAttributeBuilder().SetOid("cn").Build(),
	"c":                NewAttributeBuilder().SetOid("c").Build(),
	"seeAlso":          NewAttributeBuilder().SetOid("seeAlso").Build(),
	"ou":               NewAttributeBuilder().SetOid("ou").Build(),
	"l":                NewAttributeBuilder().SetOid("l").Build(),
	"description":      NewAttributeBuilder().SetOid("description").Build(),
	"searchGuide":      NewAttributeBuilder().SetOid("searchGuide").Build(),
	"dc":               NewAttributeBuilder().SetOid("dc").Build(),
	"serialNumber":     NewAttributeBuilder().SetOid("serialNumber").Build(),
	"owner":            NewAttributeBuilder().SetOid("owner").Build(),
	"o":                NewAttributeBuilder().SetOid("o").Build(),
	"businessCategory": NewAttributeBuilder().SetOid("businessCategory").Build(),
	"member":           NewAttributeBuilder().SetOid("member").Build(),
}

func getAttrs(oid ...string) []*Attribute {
	attrs := []*Attribute{}
	for _, o := range oid {
		attrs = append(attrs, attributes[OID(o)])
	}
	return attrs
}

var objectClasses = map[OID]*ObjectClass{
	"top": NewObjectClassBuilder().SetOid("2.5.6.0").Build(),
}

func getObjClass(oid ...string) []*ObjectClass {
	ocs := []*ObjectClass{}
	for _, o := range oid {
		ocs = append(ocs, objectClasses[OID(o)])
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

const manyAttrDefs = `
      ( 2.5.4.41 NAME 'name'
         EQUALITY caseIgnoreMatch
         SUBSTR caseIgnoreSubstringsMatch
         SYNTAX 1.3.6.1.4.1.1466.115.121.1.15 )

	  ( 2.5.4.3 NAME 'cn'
         SUP name )

      ( 2.5.4.6 NAME 'c'
         SUP name
         SYNTAX 1.3.6.1.4.1.1466.115.121.1.11
         SINGLE-VALUE )

      ( 2.5.4.15 NAME 'businessCategory'
         EQUALITY caseIgnoreMatch
         SUBSTR caseIgnoreSubstringsMatch
         SYNTAX 1.3.6.1.4.1.1466.115.121.1.15 )

      ( 0.9.2342.19200300.100.1.25 NAME 'dc'
         EQUALITY caseIgnoreIA5Match
         SUBSTR caseIgnoreIA5SubstringsMatch
         SYNTAX 1.3.6.1.4.1.1466.115.121.1.26
         SINGLE-VALUE )
`

func TestParsesAttributes(t *testing.T) {
	nameAttr := NewAttributeBuilder().
		SetOid("2.5.4.41").
		AddNames("name").
		SetEqRule(util.Unwrap(GetMatchingRule("caseIgnoreMatch"))).
		SetSubStrRule(util.Unwrap(GetMatchingRule("caseIgnoreSubstringsMatch"))).
		SetSyntax(util.Unwrap(GetSyntax(OID("1.3.6.1.4.1.1466.115.121.1.15"))), 0).
		Build()

	exp := []*Attribute{
		nameAttr,
		NewAttributeBuilder().
			SetOid("2.5.4.3").
			AddNames("cn").
			SetSup(nameAttr).
			Build(),
		NewAttributeBuilder().
			SetOid("2.5.4.6").
			AddNames("c").
			SetSup(nameAttr).
			SetSyntax(util.Unwrap(GetSyntax(OID("1.3.6.1.4.1.1466.115.121.1.11"))), 0).
			SetSingleVal(true).
			Build(),
		NewAttributeBuilder().
			SetOid("2.5.4.15").
			AddNames("businessCategory").
			SetEqRule(util.Unwrap(GetMatchingRule("caseIgnoreMatch"))).
			SetSubStrRule(util.Unwrap(GetMatchingRule("caseIgnoreSubstringsMatch"))).
			SetSyntax(util.Unwrap(GetSyntax(OID("1.3.6.1.4.1.1466.115.121.1.15"))), 0).
			Build(),
		NewAttributeBuilder().
			SetOid("0.9.2342.19200300.100.1.25").
			AddNames("dc").
			SetEqRule(util.Unwrap(GetMatchingRule("caseIgnoreIA5Match"))).
			SetSubStrRule(util.Unwrap(GetMatchingRule("caseIgnoreIA5SubstringsMatch"))).
			SetSyntax(util.Unwrap(GetSyntax(OID("1.3.6.1.4.1.1466.115.121.1.26"))), 0).
			SetSingleVal(true).
			Build(),
	}

	parsed_attrs, err := ParseAttributes(strings.NewReader(manyAttrDefs))
	if err != nil {
		t.Fatalf("Failed to parse ldif:\n%s\nErr is: %s", manyAttrDefs, err)
	}

	if len(parsed_attrs) != len(exp) {
		t.Fatalf("parsed wrong number attributes, got %d expected %d", len(parsed_attrs), len(exp))
	}
	for _, e := range exp {
		attr, ok := parsed_attrs[e.Oid()]
		if !ok {
			t.Fatalf("Parsed attributes does not contain %s", e.Oid())
		}

		if attr == nil {
			t.Fatalf("attr is nil for %s", e.Oid())
		}

		if err = AttributesAreEqual(attr, e); err != nil {
			t.Fatalf("Parsed attr did not match exp for ldif:\n%s\nGot: %s\nExp: %s\nReason: %s",
				manyAttrDefs,
				attr.String(),
				e.String(),
				err,
			)
		}
	}

	t.Logf("passed parsed ldif: %s", manyAttrDefs)
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
		exp  []*ObjectClass
	}{
		{
			ldif: `( 2.5.6.0.1 NAME 'top2' DESC 'this is some desc' )`,
			exp: []*ObjectClass{NewObjectClassBuilder().
				SetOid("2.5.6.0.1").
				AddName("top2").
				SetDesc("this is some desc").
				Build(),
			},
		},
		{
			ldif: `( 2.5.6.0.1 NAME 'top1' DESC 'somethign' )
( 2.5.6.0.2 NAME 'top2')`,
			exp: []*ObjectClass{
				NewObjectClassBuilder().SetOid("2.5.6.0.1").AddName("top1").SetDesc("somethign").Build(),
				NewObjectClassBuilder().SetOid("2.5.6.0.2").AddName("top2").Build(),
			},
		},
		{
			ldif: `( 2.5.6.6 NAME ( 'person' 'ps' ) )`,
			exp: []*ObjectClass{
				NewObjectClassBuilder().SetOid("2.5.6.6").AddName("person", "ps").Build(),
			},
		},
		{
			ldif: `( 2.5.6.6 NAME ( 'person' 'ps' ) )`,
			exp: []*ObjectClass{
				NewObjectClassBuilder().SetOid("2.5.6.6").AddName("person", "ps").Build(),
			},
		},
		{
			ldif: manyOcDefs,
			exp: []*ObjectClass{
				NewObjectClassBuilder().
					SetOid("2.5.6.11").
					AddName("applicationProcess").
					AddSup(getObjClass("top")...).
					SetKind(Structural).
					AddMustAttr(getAttrs("cn")...).
					AddMayAttr(getAttrs("seeAlso", "ou", "l", "description")...).
					Build(),
				NewObjectClassBuilder().
					SetOid("2.5.6.2").
					AddName("country").
					AddSup(getObjClass("top")...).
					SetKind(Structural).
					AddMustAttr(getAttrs("c")...).
					AddMayAttr(getAttrs("searchGuide", "description")...).
					Build(),
				NewObjectClassBuilder().
					SetOid("1.3.6.1.4.1.1466.344").
					AddName("dcObject").
					AddSup(getObjClass("top")...).
					SetKind(Auxiliary).
					AddMustAttr(getAttrs("dc")...).
					Build(),
				NewObjectClassBuilder().
					SetOid("2.5.6.14").
					AddName("device").
					AddSup(getObjClass("top")...).
					SetKind(Structural).
					AddMustAttr(getAttrs("cn")...).
					AddMayAttr(getAttrs("serialNumber", "seeAlso", "owner", "ou", "o", "l", "description")...).
					Build(),
				NewObjectClassBuilder().
					SetOid("2.5.6.9").
					AddName("groupOfNames").
					AddSup(getObjClass("top")...).
					SetKind(Structural).
					AddMustAttr(getAttrs("member", "cn")...).
					AddMayAttr(getAttrs("businessCategory", "seeAlso", "owner", "ou", "o", "description")...).
					Build(),
			},
		},
	}

	for _, test := range tests {
		parsed_objectClasses, err := ParseObjectClasses(strings.NewReader(test.ldif), attributes)
		if err != nil {
			t.Fatalf("Failed to parse ldif:\n%s\nErr is: %s", test.ldif, err)
		}

		if len(parsed_objectClasses) != len(test.exp) {
			t.Fatalf(
				"parsed wrong number of object classes, got %d expected %d",
				len(parsed_objectClasses),
				len(test.exp),
			)
		}

		for _, e := range test.exp {
			objClass, ok := parsed_objectClasses[e.Oid()]
			if !ok {
				t.Fatalf("Parsed object classes does not contain %s", e.Oid())
			}

			if err = ObjectClassesAreEqual(objClass, e); err != nil {
				t.Fatalf(
					`Parsed object class did not match expected for ldif:
%s
Got: %s
Exp: %s
Reason: %s`,
					test.ldif,
					objClass,
					e,
					err,
				)
			}
		}

		t.Logf("passed parsed ldif: %s", test.ldif)
	}
}
