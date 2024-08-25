package main

import (
	"fmt"
	"strings"
)

type EntryService interface {
	AddEntry(ar *AddRequest) (Entry, error)
}

type EntryServiceImpl struct {
	schemaService SchemaService
	entryRepo     EntryRepo
}

func NewEntryService(schemaService SchemaService, entryRepo EntryRepo) EntryService {
	return &EntryServiceImpl{schemaService, entryRepo}
}

func (e *EntryServiceImpl) findByDN(dn DN) (Entry, bool, error) {
	root, err := e.entryRepo.RootEntry()
	if err != nil {
		return Entry{}, false, err
	}

	return e.findByDnRecursive(root, dn)
}

func (e *EntryServiceImpl) findByDnRecursive(entry Entry, dn DN) (Entry, bool, error) {
	if len(dn) == 1 {
		if matchesRDN(entry, dn[0]) {
			return entry, true, nil
		} else {
			return Entry{}, false, nil
		}
	}

	// if the current top rdn does not match the current entry (working backwards)
	currRdn := dn[len(dn)-1]
	if !matchesRDN(entry, currRdn) {
		return Entry{}, false, nil
	}

	for childId := range entry.children {
		child, err := e.entryRepo.GetEntry(childId)
		if err != nil {
			return Entry{}, false, err
		}

		e, found, err := e.findByDnRecursive(child, dn[:len(dn)-1])
		if err != nil {
			return Entry{}, false, err
		} else if found {
			return e, found, nil
		}
	}

	return Entry{}, false, nil
}

func matchesRDN(e Entry, rdn RDN) bool {
	for _, ava := range rdn {
		attr, ok := e.attrs[ava.oid]
		if !ok {
			return false
		}

		if _, ok := attr[ava.val]; !ok {
			return false
		}
	}

	return true
}

func (e *EntryServiceImpl) AddEntry(ar *AddRequest) (Entry, error) {
	dn, err := e.schemaService.NormaliseDN(ar.Entry)
	if err != nil {
		return Entry{}, err
	}

	// make sure entry does not exist
	_, found, err := e.findByDN(dn)
	if err != nil {
		return Entry{}, err
	}
	if found {
		return Entry{}, EntryAleadyExistsError{dn: dn, msg: "Entry already exists"}
	}

	parent, found, err := e.findByDN(dn.parentDN())
	if err != nil {
		return Entry{}, err
	}
	if !found {
		//TODO figure out the correct matchedDN
		return Entry{}, EntryNotFoundError{dn: dn.parentDN(), msg: `Could not find entry parent.

Note - the matched DN in this error does not conform to the spec`}
	}

	objClasses, err := e.schemaService.NormaliseObjClasses(ar.Attributes)
	if err != nil {
		return Entry{}, err
	}

	attrs, err := e.schemaService.NormaliseAttrs(ar.Attributes)
	if err != nil {
		return Entry{}, err
	}

	entry := Entry{
		parent:     parent.id,
		children:   map[ID]bool{},
		objClasses: objClasses,
		attrs:      attrs,
	}

	entry, err = e.entryRepo.Save(entry)
	if err != nil {
		return Entry{}, err
	}

	parent.children[entry.id] = true
	_, err = e.entryRepo.Save(parent)
	if err != nil {
		return Entry{}, err
	}

	return entry, nil
}

type SchemaService interface {
	FindObjClassByName(name string) (ObjectClass, bool, error)
	FindAttrByName(name string) (Attribute, bool, error)
	NormaliseDN(s string) (DN, error)
	NormaliseObjClasses(attributes map[string]map[string]bool) (map[OID]bool, error)
	NormaliseAttrs(attributes map[string]map[string]bool) (map[OID]map[string]bool, error)
}

type SchemaServiceImpl struct {
	schemaRepo SchemaRepo
}

func NewSchemaService(schemaRepo SchemaRepo) SchemaService {
	return &SchemaServiceImpl{schemaRepo}
}

func (s *SchemaServiceImpl) FindObjClassByName(name string) (ObjectClass, bool, error) {
	return s.schemaRepo.FindObjClassByName(name)
}

func (s *SchemaServiceImpl) FindAttrByName(name string) (Attribute, bool, error) {
	return s.schemaRepo.FindAttributeByName(name)
}

func (s *SchemaServiceImpl) NormaliseDN(str string) (DN, error) {
	dn := DN{}

	// TODO multivalues with same key (if bothered)
	for _, rdnStr := range strings.Split(str, ",") {
		rdn := RDN{}
		for _, avaStr := range strings.Split(rdnStr, "+") {
			ava := strings.Split(avaStr, "=")

			if len(ava) != 2 {
				return dn, InvalidDNSyntaxError{dn: str, msg: fmt.Sprintf("%q is not AVA syntax", avaStr)}
			}

			attr, found, err := s.FindAttrByName(ava[0])
			if err != nil {
				return DN{}, err
			} else if !found {
				return DN{}, UndefinedAttributeTypeError{attr: ava[0], msg: err.Error()}
			}

			rdn = append(rdn, AVA{attr.numericoid, ava[1]})
		}

		dn = append(dn, rdn)
	}

	return dn, nil
}

func (s *SchemaServiceImpl) NormaliseObjClasses(attributes map[string]map[string]bool) (map[OID]bool, error) {
	objClasses := map[OID]bool{}

	attrClasses, ok := attributes["objectClass"]
	if !ok {
		return map[OID]bool{}, nil
	}

	for name := range attrClasses {
		objClass, found, err := s.schemaRepo.FindObjClassByName(name)
		if err != nil {
			return nil, err
		} else if !found {
			return nil, UndefinedAttributeTypeError{attr: name, msg: fmt.Sprintf("Could not find object class %q in shema", name)}
		}

		objClasses[objClass.numericoid] = true
	}

	return objClasses, nil
}

func (s *SchemaServiceImpl) NormaliseAttrs(attributes map[string]map[string]bool) (map[OID]map[string]bool, error) {
	attrs := map[OID]map[string]bool{}

	for key, vals := range attributes {
		if key == "objectClass" {
			continue
		}

		attr, found, err := s.schemaRepo.FindAttributeByName(key)
		if err != nil {
			return nil, err
		} else if !found {
			return nil, UndefinedAttributeTypeError{attr: key, msg: fmt.Sprintf("Could not find attribute %q in shema", key)}
		}

		attrs[attr.numericoid] = vals
	}

	return attrs, nil
}
