package main

import (
	"fmt"
	"math/rand"
)

const (
	NO_ID   ID = 0
	ROOT_ID ID = 1
)

type EntryRepo interface {
	RootEntry() (Entry, error)
	GetEntry(id ID) (Entry, error)
	Save(e Entry) (Entry, error)
}

type MemEntryRepo struct {
	entries map[ID]Entry
}

func NewMemEntryRepo() *MemEntryRepo {
	entries := map[ID]Entry{
		ROOT_ID: {id: ROOT_ID},
	}

	return &MemEntryRepo{entries}
}

func (e *MemEntryRepo) RootEntry() (Entry, error) {
	return e.entries[ROOT_ID], nil
}

func (e *MemEntryRepo) GetEntry(id ID) (Entry, error) {
	entry, ok := e.entries[id]
	if !ok {
		return Entry{}, fmt.Errorf("Could not get entry with id %q", id)
	}

	return entry, nil
}

func (e *MemEntryRepo) Save(entry Entry) (Entry, error) {
	if entry.id == NO_ID {
		entry.id = randID()
	}

	e.entries[entry.id] = entry
	return entry, nil
}

func randID() ID {
	id := ID(0)
	for id == NO_ID || id == ROOT_ID {
		id = ID(rand.Uint64())
	}
	return id
}

type SchemaRepo interface {
	GetObjClass(oid OID) (ObjectClass, error)
	GetAttribute(oid OID) (Attribute, error)
	FindObjClassByName(name string) (ObjectClass, bool, error)
	FindAttributeByName(name string) (Attribute, bool, error)
}

type MemSchemaRepo struct {
	objClasses map[OID]ObjectClass
	attribues  map[OID]Attribute
}

func NewMemSchemaRepo() *MemSchemaRepo {
	return &MemSchemaRepo{
		objClasses: map[OID]ObjectClass{},
		attribues:  map[OID]Attribute{},
	}
}

func (s *MemSchemaRepo) GetObjClass(oid OID) (ObjectClass, error) {
	objClass, ok := s.objClasses[oid]
	if !ok {
		return ObjectClass{}, fmt.Errorf("Could not find obj class with oid %q", oid)
	}

	return objClass, nil
}

func (s *MemSchemaRepo) GetAttribute(oid OID) (Attribute, error) {
	attr, ok := s.attribues[oid]
	if !ok {
		return Attribute{}, fmt.Errorf("Could not find attr with oid %q", oid)
	}

	return attr, nil
}

func (s *MemSchemaRepo) FindObjClassByName(name string) (ObjectClass, bool, error) {
	for _, objClass := range s.objClasses {
		if _, ok := objClass.names[name]; ok {
			return objClass, true, nil
		}
	}

	return ObjectClass{}, false, nil
}

func (s *MemSchemaRepo) FindAttributeByName(name string) (Attribute, bool, error) {
	for _, attr := range s.attribues {
		if _, ok := attr.names[name]; ok {
			return attr, true, nil
		}
	}

	return Attribute{}, false, nil
}
