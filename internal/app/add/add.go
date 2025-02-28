package add

import (
	"fmt"
	"log"

	"github.com/georgib0y/relientldap/internal/app/domain/dit"
)

type Normaliser interface {
	NormaliseDN(dn string) (dit.DN, error)
	NormaliseAttrs(attrs map[string]map[string]bool) (map[dit.OID]map[string]bool, error)
	NormaliseObjClasses(objClasses map[string]bool) (map[dit.OID]bool, error)
}

type Validator interface {
	ValidateEntry(dit.Entry) error
}

type entryRepo interface {
	FindByDN(dit.DN) (dit.Entry, bool, error)
	Save(dit.Entry) (dit.Entry, error)
}

type AddRequestService struct {
	n Normaliser
	v Validator
	r entryRepo
}

func (a *AddRequestService) normaliseObjClasses(ar AddRequest) (map[dit.OID]bool, error) {
	objs, ok := ar.Attributes["objectClass"]
	if !ok {
		return nil, fmt.Errorf("no object classes specified")
	}
	return a.n.NormaliseObjClasses(objs)
}

func (a *AddRequestService) normaliseAttributes(ar AddRequest) (map[dit.OID]map[string]bool, error) {
	attrs := ar.Attributes
	delete(attrs, "objectClass")

	return a.n.NormaliseAttrs(attrs)

}

// splits an add request attributes into object classes and other attrs
func (a *AddRequest) splitAttrs(ar AddRequest) (map[string]bool, map[string]map[string]bool) {
	attrs := ar.Attributes
	objs := map[string]bool{}

	for a := range attrs["objectClass"] {
		objs[a] = true
	}

	delete(attrs, "objectClass")

	return objs, attrs
}

func (a *AddRequestService) AddEntry(ar AddRequest) (dit.Entry, error) {
	log.Panicln("unimplemented")
	// dn, err := a.n.NormaliseDN(ar.Entry)
	// if err != nil {
	// 	return dit.Entry{}, err
	// }

	// _, found, err := a.r.FindByDN(dn)
	// if err != nil {
	// 	return dit.Entry{}, err
	// }

	// if found {
	// 	return dit.Entry{}, fmt.Errorf("entry already exists")
	// }

	// parent, found, err := a.r.FindByDN(dn.ParentDN())
	// if err != nil {
	// 	return dit.Entry{}, err
	// }

	// if !found {
	// 	return dit.Entry{}, fmt.Errorf("could not find entry parent")
	// }

	// objClasses, err := a.normaliseObjClasses(ar)
	// if err != nil {
	// 	return dit.Entry{}, err
	// }

	// attrs, err := a.normaliseAttributes(ar)
	// if err != nil {
	// 	return dit.Entry{}, err
	// }

	// entry := dit.Entry{
	// 	Parent:     parent.Id,
	// 	Children:   map[dit.ID]bool{},
	// 	ObjClasses: objClasses,
	// 	Attrs:      attrs,
	// }

	// if err = a.v.ValidateEntry(entry); err != nil {
	// 	return dit.Entry{}, err
	// }

	// entry, err = a.r.Save(entry)
	// if err != nil {
	// 	return dit.Entry{}, err
	// }

	// parent.Children[entry.Id] = true
	// _, err = a.r.Save(parent)

	// return entry, err
	return dit.Entry{}, nil
}
