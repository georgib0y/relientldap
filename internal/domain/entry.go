package model

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/georgib0y/relientldap/internal/util"
)

type (
	ObjClassMap map[*ObjectClass]struct{}
	AttrMap     map[*Attribute]map[string]struct{}
)

type Entry struct {
	dn         DN
	objClasses ObjClassMap
	attrs      AttrMap
}

// TODO better validation when creating entrys - ie must have at least one object class

type EntryOption func(*Entry)

func WithDN(dn DN) EntryOption {
	return func(e *Entry) {
		e.dn = dn
	}
}

func WithObjClass(oc ...*ObjectClass) EntryOption {
	return func(e *Entry) {
		for _, o := range oc {
			e.objClasses[o] = struct{}{}
		}
	}
}

func WithEntryAttr(attr *Attribute, val ...string) EntryOption {
	return func(e *Entry) {
		e.AddAttr(attr, val...)
	}
}

// TODO do i require dn when entry is made?
func NewEntry(options ...EntryOption) *Entry {
	e := &Entry{
		objClasses: ObjClassMap{},
		attrs:      AttrMap{},
	}

	for _, o := range options {
		o(e)
	}

	return e
}

func (e *Entry) Clone() *Entry {
	return &Entry{
		dn:         e.dn.Clone(),
		objClasses: util.CloneMap(e.objClasses),
		attrs:      util.CloneMapNested(e.attrs),
	}
}

// Assumes that the caller knows what their doing, and that they won't
// violate any DIT rules e.g. singleval. Required by Modify Operation,
// which allows for the entry to be temporarlily invalid
func (e *Entry) AddAttr(attr *Attribute, val ...string) {
	for _, v := range val {
		aVals, ok := e.attrs[attr]
		if !ok {
			aVals = map[string]struct{}{}
		}

		aVals[v] = struct{}{}
		e.attrs[attr] = aVals
	}
}

func (e *Entry) AddAttrSafe(attr *Attribute, val ...string) error {
	if !attr.SingleVal() {
		e.AddAttr(attr, val...)
		return nil
	}

	if len(val) != 1 {
		return fmt.Errorf("trying to add %d attributes to single val attr %s", len(val), attr.Oid())
	}
	e.attrs[attr] = map[string]struct{}{val[0]: {}}
	return nil
}

func (e *Entry) ContainsAttrVal(attr *Attribute, val string) (bool, error) {
	a, ok := e.attrs[attr]
	if !ok {
		return false, nil
	}

	matched := false
	var undefined error
	for v := range a {
		eq, ok := attr.EqRule()
		if !ok {
			return false, NewLdapError(InappropriateMatching, "", "attr %s does not have an eq rule", attr.Oid())
		}
		m, err := eq.Match(val, v)
		if err != nil {
			if errors.Is(err, UndefinedMatch) {
				undefined = err
			} else {
				return false, err
			}
		}

		if m {
			matched = true
		}
	}

	return matched, undefined
}

// Returns true if the ava was deleted or false if it could not be found
func (e *Entry) RemoveAttrVal(attr *Attribute, val string) error {
	a, ok := e.attrs[attr]
	if !ok {
		return fmt.Errorf("could not find attr %s to remove", attr.Oid())
	}

	if _, ok := a[val]; !ok {
		return fmt.Errorf("could not find value %s to remove", val)
	}

	delete(a, val)

	if len(a) == 0 {
		delete(e.attrs, attr)
	}

	return nil
}

func (e *Entry) RemoveAttrVals(attr *Attribute) bool {
	log.Print(e.attrs)
	if _, ok := e.attrs[attr]; !ok {
		return false
	}
	delete(e.attrs, attr)
	return true
}

func (e *Entry) SetRDN(rdn RDN, deleteOld bool) error {
	currRdn := e.dn.GetRDN()

	// do nothing if the rdns are the same
	if CompareRDNs(currRdn, &rdn) {
		return nil
	}

	// add any new attributes from the rdn into entry
	for a, v := range rdn.avas {
		contains, err := e.ContainsAttrVal(a, v)
		if err != nil {
			return err
		}

		if contains {
			continue
		}

		if err = e.AddAttrSafe(a, v); err != nil {
			return err
		}
	}

	if deleteOld {
		for attr, val := range currRdn.avas {
			if err := e.RemoveAttrVal(attr, val); err != nil {
				return err
			}
		}
	}

	*currRdn = rdn
	return nil
}

func (e *Entry) MatchesRdn(rdn RDN) (bool, error) {
	for attr, val := range rdn.avas {
		contains, err := e.ContainsAttrVal(attr, val)
		if errors.Is(err, UndefinedMatch) {
			return false, nil
		} else if err != nil {
			return false, err
		}

		if !contains {
			return false, nil
		}
	}

	return true, nil
}

type ChangeOperation func(*Entry) error

func AddOperation(attr *Attribute, vals ...string) ChangeOperation {
	return func(e *Entry) error {
		for _, val := range vals {
			e.AddAttr(attr, val)
		}

		return nil
	}
}

func DeleteOperation(attr *Attribute, vals ...string) ChangeOperation {
	return func(e *Entry) error {
		if len(vals) == 0 {
			e.RemoveAttrVals(attr)
			return nil
		}

		for _, val := range vals {
			e.RemoveAttrVal(attr, val)
		}

		return nil
	}
}

func ReplaceOperation(attr *Attribute, vals ...string) ChangeOperation {
	return func(e *Entry) error {
		// do nothing if the attribue does not exist
		if !e.RemoveAttrVals(attr) {
			log.Printf("replace attr does not exist: \"%s\"", attr.Oid())
			return nil
		}

		for _, val := range vals {
			e.AddAttr(attr, val)
		}

		return nil
	}
}

func (e Entry) String() string {
	sb := strings.Builder{}

	sb.WriteString("Entry: \n")
	for attr, vals := range e.attrs {
		sb.WriteString(fmt.Sprintf("\tAttr: %s\n", attr.Oid()))
		for val := range vals {
			sb.WriteString(fmt.Sprintf("\t\t%s\n", val))
		}
	}

	return sb.String()
}
