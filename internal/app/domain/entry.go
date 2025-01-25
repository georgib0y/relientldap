package domain

type Entry struct {	
	attrs      map[OID]map[string]bool
}

func (e Entry) MatchesRdn(rdn RDN) bool {
	for _, ava := range rdn.avas {
		attr, ok := e.attrs[ava.Oid]

		if !ok {
			return false
		}

		if _, ok := attr[ava.Val]; !ok {
			return false
		}
	}

	return true
}

