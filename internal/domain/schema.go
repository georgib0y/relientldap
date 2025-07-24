package model

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

// TODO probably need name or oid
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
