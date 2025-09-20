package model

import (
	"fmt"
	"io"
)

type OID string

type SchemaObject interface {
	Oid() OID
}

type Schema struct {
	attributes map[OID]*Attribute
	objClasses map[OID]*ObjectClass
}

func NewSchema(attrs map[OID]*Attribute, objClasses map[OID]*ObjectClass) *Schema {
	return &Schema{
		attributes: attrs,
		objClasses: objClasses,
	}
}

func (s *Schema) FindAttribute(name string) (*Attribute, bool) {
	if name == "objectClass" {
		return ObjectClassAttribute, true
	}

	for _, a := range s.attributes {
		if _, ok := a.names[name]; ok {
			return a, true
		}
	}

	return nil, false
}

func (s *Schema) FindObjectClass(name string) (*ObjectClass, bool) {
	if name == "top" {
		return TopObjectClass, true
	}

	for _, o := range s.objClasses {
		if _, ok := o.names[name]; ok {
			return o, true
		}
	}

	return nil, false
}

func (s *Schema) ValidateAttributeVals(attr *Attribute, vals map[string]struct{}) error {
	if len(vals) == 0 {
		return NewLdapError(ConstraintViolation, "", "Attribute %q exists for entry but has no given values", attr.Name())
	}
	if len(vals) > 1 && attr.singleVal {
		return NewLdapError(ConstraintViolation, "", "Attribute %q requires a single val but %d are given", attr.Name(), len(vals))
	}

	if attr.noUserMod {
		return NewLdapError(ConstraintViolation, "", "Attribute %q has the no user mod flag", attr.Name())
	}

	for v := range vals {
		syntax, sLen, ok := attr.Syntax()
		if !ok {
			return NewLdapError(InvalidAttributeSyntax, "", "Attribute %q or sups have no syntax", attr.Name())
		}
		if err := syntax.Validate(v); err != nil {
			return err
		}

		if sLen > 0 && len(v) > sLen {
			return NewLdapError(ConstraintViolation, "", " value %q for attribute %q exceeds syntax len %d", v, attr.Name(), sLen)
		}
	}

	return nil
}

func (s *Schema) ValidateEntry(e *Entry) error {
	if e.structural == nil {
		return NewLdapError(ConstraintViolation, "",
			"An entry must have a structural object class",
		)
	}
	for oc := range e.auxiliary {
		if oc.kind == Abstract {
			return NewLdapError(ConstraintViolation, "",
				"An entry cannot belong directly to an abstract class",
			)
		}

		return NewLdapError(ConstraintViolation, "",
			"An entry cannot have multiple structural object classes",
		)
	}

	allMust := AllObjectClassMusts(e)
	allMay := AllObjectClassMays(e)

	for must := range AllObjectClassMusts(e) {
		vals, ok := e.attrs[must]
		if !ok {
			return NewLdapError(ConstraintViolation, "",
				"Entry does not contain must attribute %q", must.Name(),
			)
		}
		if err := s.ValidateAttributeVals(must, vals); err != nil {
			return err
		}
	}

	// check that all entry attributes are either a must or may, validate if may
	for attr, vals := range e.attrs {
		if _, ok := allMust[attr]; ok {
			continue
		}

		_, ok := allMay[attr]
		if !ok {
			return NewLdapError(ConstraintViolation, "",
				"Entry does contains unspecified attribute %q", attr.Name(),
			)
		}

		if err := s.ValidateAttributeVals(attr, vals); err != nil {
			return err
		}
	}

	// check the entry contains all the attributes in it's dn
	for attr, val := range e.dn.GetRDN().avas {
		ok, err := e.ContainsAttrVal(attr, val)
		if err != nil {
			return err
		}
		if !ok {
			return NewLdapError(Other, "", "internal error: entry does not contain all the attributes in it's DN")
		}
	}
	return nil
}

// TODO let one reader read both attributes and object classes?
func LoadSchemaFromReaders(aReader, ocReader io.Reader) (*Schema, error) {
	attrs, err := ParseAttributes(aReader)
	if err != nil {
		return nil, fmt.Errorf("could not load attributes: %w", err)
	}
	logger.Print("loaded attrs")

	ocs, err := ParseObjectClasses(ocReader, attrs)
	if err != nil {
		return nil, fmt.Errorf("could not load object classes: %w", err)
	}

	logger.Print("loaded object classes")

	return NewSchema(attrs, ocs), nil
}
