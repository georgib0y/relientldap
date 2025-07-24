package ldif

import (
	"strings"
	"testing"

	d "github.com/georgib0y/relientldap/internal/domain"
)

var attributes = map[d.OID]*d.Attribute{
	"cn":               d.NewAttributeBuilder().SetOid("cn").Build(),
	"c":                d.NewAttributeBuilder().SetOid("c").Build(),
	"seeAlso":          d.NewAttributeBuilder().SetOid("seeAlso").Build(),
	"ou":               d.NewAttributeBuilder().SetOid("ou").Build(),
	"l":                d.NewAttributeBuilder().SetOid("l").Build(),
	"description":      d.NewAttributeBuilder().SetOid("description").Build(),
	"searchGuide":      d.NewAttributeBuilder().SetOid("searchGuide").Build(),
	"dc":               d.NewAttributeBuilder().SetOid("dc").Build(),
	"serialNumber":     d.NewAttributeBuilder().SetOid("serialNumber").Build(),
	"owner":            d.NewAttributeBuilder().SetOid("owner").Build(),
	"o":                d.NewAttributeBuilder().SetOid("o").Build(),
	"businessCategory": d.NewAttributeBuilder().SetOid("businessCategory").Build(),
	"member":           d.NewAttributeBuilder().SetOid("member").Build(),
}

func getAttrs(oid ...string) []*d.Attribute {
	attrs := []*d.Attribute{}
	for _, o := range oid {
		attrs = append(attrs, attributes[d.OID(o)])
	}
	return attrs
}

var objectClasses = map[d.OID]*d.ObjectClass{
	"top": d.NewObjectClassBuilder().SetOid("2.5.6.0").Build(),
}

func getObjClass(oid ...string) []*d.ObjectClass {
	ocs := []*d.ObjectClass{}
	for _, o := range oid {
		ocs = append(ocs, objectClasses[d.OID(o)])
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
	nameAttr := d.NewAttributeBuilder().
		SetOid("2.5.4.41").
		AddNames("name").
		SetEqRule(d.GetMatchingRuleUnchecked("caseIgnoreMatch")).
		SetSubStrRule(d.GetMatchingRuleUnchecked("caseIgnoreSubstringsMatch")).
		SetSyntax("1.3.6.1.4.1.1466.115.121.1.15", 0).
		Build()

	exp := []*d.Attribute{
		nameAttr,
		d.NewAttributeBuilder().
			SetOid("2.5.4.3").
			AddNames("cn").
			SetSup(nameAttr).
			Build(),
		d.NewAttributeBuilder().
			SetOid("2.5.4.6").
			AddNames("c").
			SetSup(nameAttr).
			SetSyntax("1.3.6.1.4.1.1466.115.121.1.11", 0).
			SetSingleVal(true).
			Build(),
		d.NewAttributeBuilder().
			SetOid("2.5.4.15").
			AddNames("businessCategory").
			SetEqRule(d.GetMatchingRuleUnchecked("caseIgnoreMatch")).
			SetSubStrRule(d.GetMatchingRuleUnchecked("caseIgnoreSubstringsMatch")).
			SetSyntax("1.3.6.1.4.1.1466.115.121.1.15", 0).
			Build(),
		d.NewAttributeBuilder().
			SetOid("0.9.2342.19200300.100.1.25").
			AddNames("dc").
			SetEqRule(d.GetMatchingRuleUnchecked("caseIgnoreIA5Match")).
			SetSubStrRule(d.GetMatchingRuleUnchecked("caseIgnoreIA5SubstringsMatch")).
			SetSyntax("1.3.6.1.4.1.1466.115.121.1.26", 0).
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

		if err = d.AttributesAreEqual(attr, e); err != nil {
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
		exp  []*d.ObjectClass
	}{
		{
			ldif: `( 2.5.6.0.1 NAME 'top2' DESC 'this is some desc' )`,
			exp: []*d.ObjectClass{d.NewObjectClassBuilder().
				SetOid("2.5.6.0.1").
				AddName("top2").
				SetDesc("this is some desc").
				Build(),
			},
		},
		{
			ldif: `( 2.5.6.0.1 NAME 'top1' DESC 'somethign' )
( 2.5.6.0.2 NAME 'top2')`,
			exp: []*d.ObjectClass{
				d.NewObjectClassBuilder().SetOid("2.5.6.0.1").AddName("top1").SetDesc("somethign").Build(),
				d.NewObjectClassBuilder().SetOid("2.5.6.0.2").AddName("top2").Build(),
			},
		},
		{
			ldif: `( 2.5.6.6 NAME ( 'person' 'ps' ) )`,
			exp: []*d.ObjectClass{
				d.NewObjectClassBuilder().SetOid("2.5.6.6").AddName("person", "ps").Build(),
			},
		},
		{
			ldif: `( 2.5.6.6 NAME ( 'person' 'ps' ) )`,
			exp: []*d.ObjectClass{
				d.NewObjectClassBuilder().SetOid("2.5.6.6").AddName("person", "ps").Build(),
			},
		},
		{
			ldif: manyOcDefs,
			exp: []*d.ObjectClass{
				d.NewObjectClassBuilder().
					SetOid("2.5.6.11").
					AddName("applicationProcess").
					AddSup(getObjClass("top")...).
					SetKind(d.Structural).
					AddMustAttr(getAttrs("cn")...).
					AddMayAttr(getAttrs("seeAlso", "ou", "l", "description")...).
					Build(),
				d.NewObjectClassBuilder().
					SetOid("2.5.6.2").
					AddName("country").
					AddSup(getObjClass("top")...).
					SetKind(d.Structural).
					AddMustAttr(getAttrs("c")...).
					AddMayAttr(getAttrs("searchGuide", "description")...).
					Build(),
				d.NewObjectClassBuilder().
					SetOid("1.3.6.1.4.1.1466.344").
					AddName("dcObject").
					AddSup(getObjClass("top")...).
					SetKind(d.Auxiliary).
					AddMustAttr(getAttrs("dc")...).
					Build(),
				d.NewObjectClassBuilder().
					SetOid("2.5.6.14").
					AddName("device").
					AddSup(getObjClass("top")...).
					SetKind(d.Structural).
					AddMustAttr(getAttrs("cn")...).
					AddMayAttr(getAttrs("serialNumber", "seeAlso", "owner", "ou", "o", "l", "description")...).
					Build(),
				d.NewObjectClassBuilder().
					SetOid("2.5.6.9").
					AddName("groupOfNames").
					AddSup(getObjClass("top")...).
					SetKind(d.Structural).
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

			if err = d.ObjectClassesAreEqual(objClass, e); err != nil {
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
