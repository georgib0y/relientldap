package schema

import (
	"fmt"
	"strings"

	"github.com/georgib0y/relientldap/internal/app/domain/dit"
)

type SchemaNormaliser struct {
	r Repo
}

func NewSchemaNormaliser(r Repo) *SchemaNormaliser {
	return &SchemaNormaliser{r}
}

func (n *SchemaNormaliser) NormaliseDN(s string) (dit.DN, error) {
	dn := dit.DN{}

	// TODO multivalues with same key (if bothered)
	for _, rdnStr := range strings.Split(s, ",") {
		rdn := dit.RDN{}
		for _, avaStr := range strings.Split(rdnStr, "+") {
			ava := strings.Split(avaStr, "=")

			if len(ava) != 2 {
				return dn, fmt.Errorf("%q is not AVA syntax", avaStr)

			}

			attr, found, err := n.r.FindAttrByName(ava[0])
			if err != nil {
				return dn, err
			}

			if !found {
				return dn, fmt.Errorf("couldn not find attribute %s", ava[0])
			}

			rdn.AddAVA(dit.AVA{
				Oid: attr.Numericoid,
				Val: ava[1]})
		}

		dn.AddRDN(rdn)
	}

	return dn, nil
}

func (n *SchemaNormaliser) NormaliseObjClasses(objNames map[string]bool) (map[dit.OID]bool, error) {
	objClasses := map[dit.OID]bool{}

	for name := range objNames {
		objClass, found, err := n.r.FindObjClassByName(name)
		if err != nil {
			return nil, err
		} else if !found {
			return nil, fmt.Errorf("Could not find object class %q in shema", name)
		}

		objClasses[objClass.Numericoid] = true
	}

	return objClasses, nil
}

func (n *SchemaNormaliser) NormaliseAttrs(attributes map[string]map[string]bool) (map[dit.OID]map[string]bool, error) {
	attrs := map[dit.OID]map[string]bool{}

	for key, vals := range attributes {
		if key == "objectClass" {
			continue
		}

		attr, found, err := n.r.FindAttrByName(key)
		if err != nil {
			return nil, err
		} else if !found {
			return nil, fmt.Errorf("Could not find attribute %q in shema", key)
		}

		attrs[attr.Numericoid] = vals
	}

	return attrs, nil
}
