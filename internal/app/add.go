package app

import (
	d "github.com/georgib0y/relientldap/internal/domain"
)

type AddService struct {
	schema    *d.Schema
	scheduler *Scheduler
}

func NewAddService(schema *d.Schema, scheduler *Scheduler) *AddService {
	return &AddService{schema, scheduler}
}

type AddRequest interface {
	Dn() string
	Attributes() map[string][]string
}

func (a *AddService) objectClassOpt(reqAttrs map[string][]string) ([]d.EntryOption, error) {
	// TODO does oid need to be checked as well?
	vals, ok := reqAttrs["objectClass"]
	if !ok {
		return nil, d.NewLdapError(d.ObjectClassViolation, "", "no object class was specified for entry")
	}

	opts := []d.EntryOption{}

	for _, v := range vals {
		o, ok := a.schema.FindObjectClass(v)
		if !ok {
			return nil, d.NewLdapError(d.NoSuchAttribute, "", "could not find object class with name %s", v)
		}

		switch o.Kind() {
		case d.Structural:
			opts = append(opts, d.WithStructural(o))
		case d.Auxiliary:
			opts = append(opts, d.WithAuxiliary(o))
		case d.Abstract:
			return nil, d.NewLdapError(d.ObjectClassViolation, "", "trying to add an abstract object class %s directly", o.Name())
		}
	}

	return opts, nil
}

func (a *AddService) attributeOpts(reqAttrs map[string][]string) ([]d.EntryOption, error) {
	opts := []d.EntryOption{}
	for name, vals := range reqAttrs {
		if name == "objectClass" {
			// handle ocs separately
			continue
		}
		attr, ok := a.schema.FindAttribute(name)
		if !ok {
			return nil, d.NewLdapError(d.UndefinedAttributeType, "", "unknown attribute %s", name)
		}

		opts = append(opts, d.WithEntryAttr(attr, vals...))
	}

	return opts, nil
}

func (a *AddService) AddEntry(ar AddRequest) (*d.Entry, error) {
	dn, err := d.NormaliseDN(a.schema, ar.Dn())
	if err != nil {
		return nil, err
	}

	reqAttrs := ar.Attributes()

	opts := []d.EntryOption{}
	ocs, err := a.objectClassOpt(reqAttrs)
	if err != nil {
		return nil, err
	}
	opts = append(opts, ocs...)

	attrs, err := a.attributeOpts(reqAttrs)
	if err != nil {
		return nil, err
	}
	opts = append(opts, attrs...)

	// TODO get opts
	entry, err := d.NewEntry(a.schema, dn, opts...)
	if err != nil {
		return nil, err
	}

	return entry, ScheduleAwaitError(a.scheduler, func(dit d.DIT) error {
		return dit.InsertEntry(dn, entry)
	})
}
