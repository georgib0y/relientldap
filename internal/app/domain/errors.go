package domain

import (
	"errors"
	"fmt"
)

var (
	ErrNodeNotLeaf  = errors.New("Node is not a leaf node")
	ErrUnknownScope = errors.New("Unknown scope")
)

type NodeNotFoundError struct {
	requestedDN, matchedDn DN
}

func (e NodeNotFoundError) Error() string {
	return fmt.Sprintf("requested DN: %s, matched up to: %s", e.requestedDN, e.matchedDn)
}

func (e *NodeNotFoundError) prependMatchedDn(rdn RDN) {
	e.matchedDn.rdns = append([]RDN{rdn}, e.matchedDn.rdns...)
}
