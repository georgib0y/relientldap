package model

import (
	"errors"
	"fmt"
)

var (
	ErrNodeNotLeaf  = errors.New("Node is not a leaf node")
	ErrUnknownScope = errors.New("Unknown scope")
)

type NodeNotFoundError struct {
	RequestedDN, MatchedDN DN
}

func (e NodeNotFoundError) Error() string {
	return fmt.Sprintf("requested DN: %s, matched up to: %s", e.RequestedDN, e.MatchedDN)
}

func (e *NodeNotFoundError) prependMatchedDn(rdn RDN) {
	e.MatchedDN.rdns = append([]RDN{rdn}, e.MatchedDN.rdns...)
}
