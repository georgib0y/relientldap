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

func (s *Schema) FindAttribute(name string) (*Attribute, bool) {
	for _, a := range s.attributes {
		if _, ok := a.names[name]; ok {
			return a, true
		}
	}

	return nil, false
}
