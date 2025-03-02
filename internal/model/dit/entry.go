package dit

import (
	"fmt"
	"log"
	"strings"
)

type Entry struct {
	dn         DN
	objClasses map[OID]bool
	attrs      map[OID]map[string]bool
}

// TODO better validation when creating entrys - ie must have at least one object class

type EntryOption func(*Entry)

func WithDN(dn DN) EntryOption {
	return func(e *Entry) {
		e.dn = dn
	}
}

func WithObjClass(oid ...OID) EntryOption {
	return func(e *Entry) {
		for _, o := range oid {
			e.objClasses[o] = true
		}
	}
}

func WithEntryAttr(ava ...AVA) EntryOption {
	return func(e *Entry) {
		e.AddAttr(ava...)
	}
}

// TODO do i require dn when entry is made?
func NewEntry(options ...EntryOption) Entry {
	e := Entry{
		attrs: map[OID]map[string]bool{},
	}

	for _, o := range options {
		o(&e)
	}

	return e
}

func (e Entry) Clone() Entry {
	attrs := map[OID]map[string]bool{}

	for o, attr := range e.attrs {
		vals := map[string]bool{}
		for a := range attr {
			vals[a] = true
		}
		attrs[o] = vals
	}

	return Entry{
		dn:    e.dn.Clone(),
		attrs: attrs,
	}
}

func (e *Entry) AddAttr(ava ...AVA) {
	for _, a := range ava {
		attr, ok := e.attrs[a.Oid]
		if !ok {
			attr = map[string]bool{}
		}

		attr[a.Val] = true
		e.attrs[a.Oid] = attr
	}
}

func (e Entry) ContainsAttr(ava AVA) bool {
	attr, ok := e.attrs[ava.Oid]
	if !ok {
		return false
	}

	_, ok = attr[ava.Val]
	return ok
}

// Returns true if the ava was deleted or false if it could not be found
func (e *Entry) RemoveAttr(ava AVA) bool {
	attr, ok := e.attrs[ava.Oid]
	if !ok {
		return false
	}

	if _, ok := attr[ava.Val]; !ok {
		return false
	}

	delete(attr, ava.Val)

	if len(attr) == 0 {
		delete(e.attrs, ava.Oid)
	}

	return true
}

// TODO better name?
func (e *Entry) RemoveAllAttr(oid OID) bool {
	log.Print(e.attrs)
	if _, ok := e.attrs[oid]; !ok {
		return false
	}
	delete(e.attrs, oid)
	return true
}

func (e Entry) SetRDN(rdn RDN, deleteOld bool) {
	currRdn := e.dn.GetRDN()

	// do nothing if the rdns are the same
	if CompareRDNs(currRdn, rdn) {
		return
	}

	e.dn.ReplaceRDN(rdn)

	if !deleteOld {
		return
	}

	for ava := range currRdn.avas {
		if !e.RemoveAttr(ava) {
			log.Printf("trying to delete ava: %s, from current rdn but doesnt exist!!", ava)
		}
	}
}

func (e Entry) MatchesRdn(rdn RDN) bool {
	for ava := range rdn.avas {
		if !e.ContainsAttr(ava) {
			return false
		}
	}

	return true
}

type ChangeOperation func(*Entry) error

func AddOperation(oid OID, vals ...string) ChangeOperation {
	return func(e *Entry) error {
		for _, val := range vals {
			e.AddAttr(AVA{oid, val})
		}

		return nil
	}
}

func DeleteOperation(oid OID, vals ...string) ChangeOperation {
	return func(e *Entry) error {
		if len(vals) == 0 {
			e.RemoveAllAttr(oid)
			return nil
		}

		for _, val := range vals {
			e.RemoveAttr(AVA{oid, val})
		}

		return nil
	}
}

func ReplaceOperation(oid OID, vals ...string) ChangeOperation {
	return func(e *Entry) error {
		// do nothing if the attribue does not exist
		if !e.RemoveAllAttr(oid) {
			log.Printf("replace attr does not exist: \"%s\"", oid)
			return nil
		}

		for _, val := range vals {
			e.AddAttr(AVA{oid, val})
		}

		return nil
	}
}

func (e Entry) String() string {
	sb := strings.Builder{}

	sb.WriteString("Entry: \n")
	for attr, vals := range e.attrs {
		sb.WriteString(fmt.Sprintf("\tAttr: %s\n", attr))
		for val := range vals {
			sb.WriteString(fmt.Sprintf("\t\t%s\n", val))
		}
	}

	return sb.String()
}
