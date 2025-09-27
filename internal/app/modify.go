package app

import d "github.com/georgib0y/relientldap/internal/domain"

type ModifyService struct {
	schema    *d.Schema
	scheduler *Scheduler
}

func NewModifyService(schema *d.Schema, scheduler *Scheduler) *ModifyService {
	return &ModifyService{schema, scheduler}
}

type ModifyOperation int

const (
	ModifyAdd     ModifyOperation = 0
	ModifyDelete  ModifyOperation = 1
	ModifyReplace ModifyOperation = 2
)

type Modification interface {
	ModOp() ModifyOperation
	Attribute() string
	Vals() []string
}

type ModifyRequest interface {
	Dn() string
	Modifications() []Modification
}

func (m *ModifyService) ModifyEntry(mr ModifyRequest) error {
	dn, err := d.NormaliseDN(m.schema, mr.Dn())
	if err != nil {
		return err
	}

	changes := []d.ChangeOperation{}

	for _, mod := range mr.Modifications() {
		attr, ok := m.schema.FindAttribute(mod.Attribute())
		if !ok {
			return d.NewLdapError(d.NoSuchAttribute, nil, "could not find attr: %q", mod.Attribute())
		}

		switch mod.ModOp() {
		case ModifyAdd:
			changes = append(changes, d.AddOperation(attr, mod.Vals()...))
		case ModifyDelete:
			changes = append(changes, d.DeleteOperation(attr, mod.Vals()...))
		case ModifyReplace:
			changes = append(changes, d.ReplaceOperation(attr, mod.Vals()...))
		default:
			return d.NewLdapError(d.ProtocolError, nil, "unknown modification operation type: %d", mod.ModOp())
		}
	}

	return ScheduleAwaitError(m.scheduler, func(dit d.DIT) error {
		return dit.ModifyEntry(dn, changes...)
	})
}

type ModifyDnRequest interface {
	Dn() string
	UpdatedRdn() string
	RemoveExistingRdn() bool
	NewParentDn() (string, bool)
}

func (m *ModifyService) ModifyEntryDn(mr ModifyDnRequest) error {
	dn, err := d.NormaliseDN(m.schema, mr.Dn())
	if err != nil {
		return err
	}

	newRdn, err := d.NormaliseRDN(m.schema, mr.UpdatedRdn())
	if err != nil {
		return err
	}

	var newParentDn *d.DN
	if s, ok := mr.NewParentDn(); ok {
		pdn, err := d.NormaliseDN(m.schema, s)
		if err != nil {
			return err
		}
		*newParentDn = pdn
	}

	return ScheduleAwaitError(m.scheduler, func(dit d.DIT) error {
		return dit.ModifyEntryDN(dn, newRdn, mr.RemoveExistingRdn(), newParentDn)
	})
}
