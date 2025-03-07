package schema

import "github.com/georgib0y/relientldap/internal/model/dit"

type Schema struct {
	objClasses map[dit.OID]ObjectClass
	attributes map[dit.OID]Attribute
}
