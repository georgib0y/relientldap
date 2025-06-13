package ldif

import (
	"strings"
	"testing"

	sch "github.com/georgib0y/relientldap/internal/model/schema"
)

var attributes = map[sch.OID]*sch.Attribute{
	"cn":               sch.NewAttributeBuilder().SetOid("cn").Build(),
	"c":                sch.NewAttributeBuilder().SetOid("c").Build(),
	"seeAlso":          sch.NewAttributeBuilder().SetOid("seeAlso").Build(),
	"ou":               sch.NewAttributeBuilder().SetOid("ou").Build(),
	"l":                sch.NewAttributeBuilder().SetOid("l").Build(),
	"description":      sch.NewAttributeBuilder().SetOid("description").Build(),
	"searchGuide":      sch.NewAttributeBuilder().SetOid("searchGuide").Build(),
	"dc":               sch.NewAttributeBuilder().SetOid("dc").Build(),
	"serialNumber":     sch.NewAttributeBuilder().SetOid("serialNumber").Build(),
	"owner":            sch.NewAttributeBuilder().SetOid("owner").Build(),
	"o":                sch.NewAttributeBuilder().SetOid("o").Build(),
	"businessCategory": sch.NewAttributeBuilder().SetOid("businessCategory").Build(),
	"member":           sch.NewAttributeBuilder().SetOid("member").Build(),
}

func getAttrs(oid ...string) []*sch.Attribute {
	attrs := []*sch.Attribute{}
	for _, o := range oid {
		attrs = append(attrs, attributes[sch.OID(o)])
	}
	return attrs
}

var objectClasses = map[sch.OID]*sch.ObjectClass{
	"top": sch.NewObjectClassBuilder().SetOid("2.5.6.0").Build(),
}

func getObjClass(oid ...string) []*sch.ObjectClass {
	ocs := []*sch.ObjectClass{}
	for _, o := range oid {
		ocs = append(ocs, objectClasses[sch.OID(o)])
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
	nameAttr := sch.NewAttributeBuilder().
		SetOid("2.5.4.41").
		AddNames("name").
		SetEqRule(sch.GetMatchingRuleUnchecked("caseIgnoreMatch")).
		SetSubStrRule(sch.GetMatchingRuleUnchecked("caseIgnoreSubstringsMatch")).
		SetSyntax("1.3.6.1.4.1.1466.115.121.1.15", 0).
		Build()

	exp := []*sch.Attribute{
		nameAttr,
		sch.NewAttributeBuilder().
			SetOid("2.5.4.3").
			AddNames("cn").
			SetSup(nameAttr).
			Build(),
		sch.NewAttributeBuilder().
			SetOid("2.5.4.6").
			AddNames("c").
			SetSup(nameAttr).
			SetSyntax("1.3.6.1.4.1.1466.115.121.1.11", 0).
			SetSingleVal().
			Build(),
		sch.NewAttributeBuilder().
			SetOid("2.5.4.15").
			AddNames("businessCategory").
			SetEqRule(sch.GetMatchingRuleUnchecked("caseIgnoreMatch")).
			SetSubStrRule(sch.GetMatchingRuleUnchecked("caseIgnoreSubstringsMatch")).
			SetSyntax("1.3.6.1.4.1.1466.115.121.1.15", 0).
			Build(),
		sch.NewAttributeBuilder().
			SetOid("0.9.2342.19200300.100.1.25").
			AddNames("dc").
			SetEqRule(sch.GetMatchingRuleUnchecked("caseIgnoreIA5Match")).
			SetSubStrRule(sch.GetMatchingRuleUnchecked("caseIgnoreIA5SubstringsMatch")).
			SetSyntax("1.3.6.1.4.1.1466.115.121.1.26", 0).
			SetSingleVal().
			Build(),
	}

	p := NewAttributeParser()
	parsed_attrs, err := ParseReader(strings.NewReader(manyAttrDefs), p)
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

		if err = sch.AttributesAreEqual(attr, e); err != nil {
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

	( 2.5.6.0 NAME 'top' DESC 'this is some desc' )
`

func TestParsesObjectClass(t *testing.T) {
	tests := []struct {
		ldif string
		exp  []*sch.ObjectClass
	}{
		{
			ldif: `( 2.5.6.0 NAME 'top' DESC 'this is some desc' )`,
			exp: []*sch.ObjectClass{sch.NewObjectClassBuilder().
				SetOid("2.5.6.0").
				AddName("top").
				SetDesc("this is some desc").
				Build(),
			},
		},
		{
			ldif: `( 2.5.6.0 NAME 'top' DESC 'somethign' )
( 2.5.6.1 NAME 'secondTop')`,
			exp: []*sch.ObjectClass{
				sch.NewObjectClassBuilder().SetOid("2.5.6.0").AddName("top").SetDesc("somethign").Build(),
				sch.NewObjectClassBuilder().SetOid("2.5.6.1").AddName("secondTop").Build(),
			},
		},
		{
			ldif: `( 2.5.6.6 NAME ( 'person' 'ps' ) )`,
			exp: []*sch.ObjectClass{
				sch.NewObjectClassBuilder().SetOid("2.5.6.6").AddName("person", "ps").Build(),
			},
		},
		{
			ldif: `( 2.5.6.6 NAME ( 'person' 'ps' ) )`,
			exp: []*sch.ObjectClass{
				sch.NewObjectClassBuilder().SetOid("2.5.6.6").AddName("person", "ps").Build(),
			},
		},
		{
			ldif: manyOcDefs,
			exp: []*sch.ObjectClass{
				sch.NewObjectClassBuilder().
					SetOid("2.5.6.11").
					AddName("applicationProcess").
					AddSup(getObjClass("top")...).
					SetKind(sch.Structural).
					AddMustAttr(getAttrs("cn")...).
					AddMayAttr(getAttrs("seeAlso", "ou", "l", "description")...).
					Build(),
				sch.NewObjectClassBuilder().
					SetOid("2.5.6.2").
					AddName("country").
					AddSup(getObjClass("top")...).
					SetKind(sch.Structural).
					AddMustAttr(getAttrs("c")...).
					AddMayAttr(getAttrs("searchGuide", "description")...).
					Build(),
				sch.NewObjectClassBuilder().
					SetOid("1.3.6.1.4.1.1466.344").
					AddName("dcObject").
					AddSup(getObjClass("top")...).
					SetKind(sch.Auxilary).
					AddMustAttr(getAttrs("dc")...).
					Build(),
				sch.NewObjectClassBuilder().
					SetOid("2.5.6.14").
					AddName("device").
					AddSup(getObjClass("top")...).
					SetKind(sch.Structural).
					AddMustAttr(getAttrs("cn")...).
					AddMayAttr(getAttrs("serialNumber", "seeAlso", "owner", "ou", "o", "l", "description")...).
					Build(),
				sch.NewObjectClassBuilder().
					SetOid("2.5.6.9").
					AddName("groupOfNames").
					AddSup(getObjClass("top")...).
					SetKind(sch.Structural).
					AddMustAttr(getAttrs("member", "cn")...).
					AddMayAttr(getAttrs("businessCategory", "seeAlso", "owner", "ou", "o", "description")...).
					Build(),
				sch.NewObjectClassBuilder().
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

			if err = sch.ObjectClassesAreEqual(objClass, e); err != nil {
				t.Fatalf("Parsed object class did not match expected for ldif:\n%s\nGot: %s\nExp: %s\nReason: %s", test.ldif, objClass, e, err)
			}
		}

		t.Logf("passed parsed ldif: %s", test.ldif)
	}
}
