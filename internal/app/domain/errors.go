package domain

import (
	"fmt"
	"log"
)

type NodeNotFoundError struct {
	requestedDN,	matchedDn DN
}

func (e NodeNotFoundError) Error() string {
	return fmt.Sprintf("requested DN: %s, matched up to: %s", e.requestedDN, e.matchedDn)
}

func (e *NodeNotFoundError) prependMatchedDn(rdn RDN) {
	log.Println("prepending: ", rdn)
	e.matchedDn.rdns = append([]RDN{rdn}, e.matchedDn.rdns...)
}
