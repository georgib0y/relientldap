package schema

import "github.com/georgib0y/gbldap/internal/app/domain"

type Repo interface {
	GetObjClass(oid domain.OID) (domain.ObjectClass, error)
	GetAttribute(oid domain.OID) (domain.Attribute, error)
	FindAttrByName(name string) (domain.Attribute, bool, error)
	FindObjClassByName(name string) (domain.ObjectClass, bool, error)
}
