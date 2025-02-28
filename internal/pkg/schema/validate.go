package schema

import (
	"fmt"
	"log"

	"github.com/georgib0y/relientldap/internal/app/domain/dit"
	"github.com/georgib0y/relientldap/internal/app/domain/schema"
)

type ObjectClassMap map[dit.OID]schema.ObjectClass
type AttributeMap map[dit.OID]schema.Attribute

type SchemaValidator struct {
	r Repo
}

func (v *SchemaValidator) getAllObjecClasses(oids map[dit.OID]bool) (ObjectClassMap, error) {
	objClasses := ObjectClassMap{}

	for oid := range oids {
		objClass, err := v.r.GetObjClass(oid)
		if err != nil {
			return nil, err
		}
		objClasses[oid] = objClass
	}

	return objClasses, nil
}

func (v *SchemaValidator) validateStructuralCount(objClasses ObjectClassMap) error {
	count := 0
	for _, objClass := range objClasses {
		if objClass.Kind == schema.Structural {
			count++
		}
	}

	if count != 1 {
		return fmt.Errorf("Invalid number of structural object classes: %d", count)
	}

	return nil
}

func (v *SchemaValidator) getAllRequiredAttrs(objClasses ObjectClassMap) (AttributeMap, error) {
	must := AttributeMap{}

	for _, objClass := range objClasses {
		for oid := range objClass.MustAttrs {
			attr, err := v.r.GetAttribute(oid)
			if err != nil {
				return nil, err
			}

			must[oid] = attr
		}
	}

	return must, nil
}

func (v *SchemaValidator) getAllOptionalAttrs(objClasses ObjectClassMap) (AttributeMap, error) {
	may := AttributeMap{}

	for _, objClass := range objClasses {
		for oid := range objClass.MayAttrs {
			attr, err := v.r.GetAttribute(oid)
			if err != nil {
				return nil, err
			}

			may[oid] = attr
		}
	}

	return may, nil
}

func (v *SchemaValidator) validateRequiredAttr(entry dit.Entry, attr schema.Attribute) error {
	// TODO more thorough attribute validation
	log.Panicln("unimplemented, entry Attrs")
	// if _, ok := entry.Attrs[attr.Numericoid]; !ok {
	// 	return fmt.Errorf("Attribute %s is required", attr.Numericoid)
	// }

	return nil
}

func (v *SchemaValidator) findUnspecifiedAttrs(entry dit.Entry, reqAttrs, optAttrs AttributeMap) error {
	log.Panicln("unimplemented, entry Attrs")
	// for oid := range entry.Attrs {
	// 	_, okReq := reqAttrs[oid]
	// 	_, okOpt := optAttrs[oid]

	// 	if !okReq && !okOpt {
	// 		return fmt.Errorf("%s not a part of any required or optional attributes", oid)
	// 	}
	// }

	return nil
}

func (v *SchemaValidator) validateAttr(entry dit.Entry, attr schema.Attribute, required bool) error {
	log.Panicln("unimplemented, entry Attrs")
	// vals, ok := entry.Attrs[attr.Numericoid]
	// if !ok {
	// 	if required {
	// 		return fmt.Errorf("attr %s is required for entry %d", attr.Numericoid, entry.Id)
	// 	} else {
	// 		return nil
	// 	}
	// }

	// if len(vals) == 0 {
	// 	return fmt.Errorf("values empty in entry %d for attr %s", entry.Id, attr.Numericoid)
	// }

	// if attr.SingleVal && len(vals) > 1 {
	// 	return fmt.Errorf("attr %s is signleval but entry %d has multiple values", attr.Numericoid, entry.Id)
	// }

	// TODO more attr validation

	return nil
}

func (s *SchemaValidator) validateAttrs(entry dit.Entry, objClasses ObjectClassMap) error {
	reqAttrs, err := s.getAllRequiredAttrs(objClasses)
	if err != nil {
		return err
	}

	for _, req := range reqAttrs {
		if err = s.validateRequiredAttr(entry, req); err != nil {
			return err
		}

		if err = s.validateAttr(entry, req, true); err != nil {
			return err
		}
	}

	optAttrs, err := s.getAllOptionalAttrs(objClasses)
	if err != nil {
		return err
	}

	if err = s.findUnspecifiedAttrs(entry, reqAttrs, optAttrs); err != nil {
		return err
	}

	for _, opt := range optAttrs {
		if err = s.validateAttr(entry, opt, false); err != nil {
			return err
		}
	}

	return nil
}

func (s *SchemaValidator) ValidateEntry(entry dit.Entry) error {
	log.Panicln("unimplemented, entry ObjClasses")
	return nil
	// objClasses, err := s.getAllObjecClasses(entry.ObjClasses)
	// if err != nil {
	// 	return err
	// }

	// if err := s.validateStructuralCount(objClasses); err != nil {
	// 	return err
	// }

	// return s.validateAttrs(entry, objClasses)
}
