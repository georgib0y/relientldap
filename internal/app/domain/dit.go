package domain

import (
	"errors"
	"log"
	"reflect"
)

type DITNode struct {
	parent *DITNode
	children []*DITNode
	entry Entry
}

type DIT struct {
	root *DITNode
}

func (d DIT) GetEntry(dn DN) (Entry, error) {
	log.Printf("dn is %s", dn)

	node, err := getNode(dn.rdns, d.root)

	var nfErr *NodeNotFoundError
	if errors.As(err, &nfErr) {
		nfErr.requestedDN = dn
		return Entry{}, nfErr
	} else if err != nil {
		return Entry{}, err
	}

	return node.entry, nil
}

func (d DIT) InsertEntry(dn DN, entry Entry) error {
	pDn := DN{dn.rdns[:len(dn.rdns)-1]}
	pNode, err := getNode(pDn.rdns, d.root)

	var nfErr *NodeNotFoundError
	if errors.As(err, &nfErr) {
		nfErr.requestedDN = dn
		return nfErr
	} else if err != nil {
		return err
	}

	

	node.entry = entry;
	return nil
}

func getNode(rdns []RDN, node *DITNode) (*DITNode, error) {
	log.Printf("at %s", rdns[0])
	if !node.entry.MatchesRdn(rdns[0]) {
		log.Printf("rdn %s did not match entry", rdns[0])
		return nil, &NodeNotFoundError{}
	}
	
	if len(rdns) == 1 {
		return node, nil
	}

	var finalErr error
	var nfErr *NodeNotFoundError
	
	for _, c := range node.children {		
		n, err := getNode(rdns[1:], c)

		if err == nil{
			return n, nil
		}

		if !errors.As(err, &nfErr) {
			log.Println("err is not notfounderr: ", reflect.TypeOf(err))
			return nil, err
		}

		finalErr = err
	}

	// at this point err can only be not found
	_ = errors.As(finalErr, &nfErr)

	// prepend this rdn to the matched rdn
	nfErr.prependMatchedDn(rdns[0])
	return nil, nfErr
}




