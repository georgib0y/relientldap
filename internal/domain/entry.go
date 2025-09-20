package model

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/georgib0y/relientldap/internal/util"
)

type Entry struct {
	dn         DN
	structural *ObjectClass
	auxiliary  map[*ObjectClass]struct{}
	attrs      map[*Attribute]map[string]struct{}
}

type EntryOption func(*Entry)

func WithStructural(s *ObjectClass) EntryOption {
	return func(e *Entry) {
		e.structural = s
	}
}

func WithAuxiliary(aux ...*ObjectClass) EntryOption {
	// TODO check and shake dependencies
	return func(e *Entry) {
		for _, oc := range aux {
			e.auxiliary[oc] = struct{}{}
		}
	}
}

func WithEntryAttr(attr *Attribute, val ...string) EntryOption {
	return func(e *Entry) {
		e.AddAttr(attr, val...)
	}
}

func NewEntry(schema *Schema, dn DN, options ...EntryOption) (*Entry, error) {
	e := &Entry{
		dn:        dn,
		auxiliary: map[*ObjectClass]struct{}{},
		attrs:     map[*Attribute]map[string]struct{}{},
	}

	for _, o := range options {
		o(e)
	}

	// include DN attributes if not already
	for attr, val := range dn.GetRDN().avas {
		e.AddAttr(attr, val)
	}

	if err := schema.ValidateEntry(e); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Entry) Dn() DN {
	return e.dn
}

func (e *Entry) Clone() *Entry {
	return &Entry{
		dn:         e.dn.Clone(),
		structural: e.structural,
		auxiliary:  util.CloneMap(e.auxiliary),
		attrs:      util.CloneMapNested(e.attrs),
	}
}

// Assumes that the caller knows what their doing, and that they won't
// violate any DIT rules e.g. singleval. Required by Modify Operation,
// which allows for the entry to be temporarlily invalid
func (e *Entry) AddAttrUnsafe(attr *Attribute, val ...string) {
	for _, v := range val {
		aVals, ok := e.attrs[attr]
		if !ok {
			aVals = map[string]struct{}{}
		}

		aVals[v] = struct{}{}
		e.attrs[attr] = aVals
	}
}

func (e *Entry) AddAttr(attr *Attribute, val ...string) error {
	if !attr.SingleVal() {
		e.AddAttrUnsafe(attr, val...)
		return nil
	}

	if len(val) != 1 {
		return fmt.Errorf("trying to add %d attributes to single val attr %s", len(val), attr.Oid())
	}
	e.attrs[attr] = map[string]struct{}{val[0]: {}}
	return nil
}

func (e *Entry) ConatinsObjectClass(objClass *ObjectClass) bool {
	if e.structural == objClass {
		return true
	}
	_, ok := e.auxiliary[objClass]
	return ok
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

		if err = e.AddAttr(a, v); err != nil {
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

func (e *Entry) String() string {
	sb := strings.Builder{}
	fmt.Fprintf(&sb, "Entry: \nStructural: %s\nAuxiliary: ", e.structural.Name())
	for oc := range e.auxiliary {
		fmt.Fprintf(&sb, " %s", oc.Name())
	}
	sb.WriteString("\nAttributes:\n")
	for attr, vals := range e.attrs {
		fmt.Fprintf(&sb, "\t%s:", attr.Name())
		for val := range vals {
			fmt.Fprintf(&sb, " %s", val)
		}
		sb.WriteRune('\n')
	}

	return sb.String()
}
