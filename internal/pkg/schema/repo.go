package schema

import (
	"github.com/georgib0y/relientldap/internal/app/domain/dit"
	"github.com/georgib0y/relientldap/internal/app/domain/schema"
)

type Repo interface {
	GetObjClass(oid dit.OID) (schema.ObjectClass, error)
	GetAttribute(oid dit.OID) (schema.Attribute, error)
	FindAttrByName(name string) (schema.Attribute, bool, error)
	FindObjClassByName(name string) (schema.ObjectClass, bool, error)
}
