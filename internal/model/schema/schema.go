package schema

type OID string

type SchemaObject interface {
	Oid() OID
}

type Schema struct {
	objClasses map[OID]*ObjectClass
	attributes map[OID]*Attribute
}
