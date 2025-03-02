package dit

import (
	"log"
)

type SearchScope int

const (
	BaseObject SearchScope = iota
	SingleLevel
	WholeSubtree
	SubordinateSubtree
)

// TODO Greater/Less or equal, Substring, Approx match, extensible match
type Filter func(Entry) bool

func FilterAnd(f1, f2 Filter) Filter {
	return func(e Entry) bool {
		return f1(e) && f2(e)
	}
}

func FilterOr(f1, f2 Filter) Filter {
	return func(e Entry) bool {
		return f1(e) || f2(e)
	}
}

func FilterNot(f Filter) Filter {
	return func(e Entry) bool {
		return !f(e)
	}
}

func NewPresenceFilter(target OID) Filter {
	return func(e Entry) bool {
		_, ok := e.attrs[target]
		return ok
	}
}

func NewEqualityFilter(target OID, val string) Filter {
	return func(e Entry) bool {
		vals, ok := e.attrs[target]
		if !ok {
			return false
		}

		_, ok = vals[val]
		return ok
	}
}

// TODO alias deref, size and time limits, types only, requested attrs
func (d DIT) Search(baseDn DN, scope SearchScope, filter Filter) ([]Entry, error) {
	node, err := d.getNode(baseDn)
	if err != nil {
		return nil, err
	}

	switch scope {
	case BaseObject:
		res := []Entry{}
		if e, ok := searchBaseObject(node.entry, filter); ok {
			res = append(res, e)
		}
		return res, nil
	case SingleLevel:
		return searchSingleLevel(node, filter), nil
	case WholeSubtree:
		return searchWholeSubtree(node, filter), nil
	case SubordinateSubtree:
		return searchSubordiateSubtree(node, filter), nil
	}

	return nil, ErrUnknownScope
}

func searchBaseObject(e Entry, filter Filter) (Entry, bool) {
	return e, filter(e)
}

func searchSingleLevel(base *DITNode, filter Filter) []Entry {
	matched := []Entry{}
	for c := range base.children {
		if filter(c.entry) {
			matched = append(matched, c.entry)
		}
	}
	return matched
}

func searchWholeSubtree(base *DITNode, filter Filter) []Entry {
	matched := []Entry{}
	WalkTree(base, func(e Entry) {
		if filter(e) {
			matched = append(matched, e)
		}
	})

	return matched
}

func searchSubordiateSubtree(base *DITNode, filter Filter) []Entry {
	matched := []Entry{}
	for c := range base.children {
		WalkTree(c, func(e Entry) {
			if filter(e) {
				matched = append(matched, e)
			}
		})
	}

	return matched
}
