package domain

import (
	"errors"
	"log"
)

type DITNode struct {
	parent   *DITNode
	children map[*DITNode]bool
	entry    Entry
}

func NewDITNode(parent *DITNode, entry Entry) *DITNode {
	return &DITNode{
		parent:   parent,
		children: map[*DITNode]bool{},
		entry:    entry,
	}
}

func (n *DITNode) AddChild(entry Entry) {
	c := NewDITNode(n, entry)
	n.children[c] = true
}

func (n *DITNode) DeleteChild(child *DITNode) {
	delete(n.children, child)
}

// TODO domain context
type DIT struct {
	root *DITNode
}

func (d DIT) GetEntry(dn DN) (Entry, error) {
	node, err := d.getNode(dn)
	if err != nil {
		return Entry{}, err
	}

	return node.entry, nil
}

func (d DIT) InsertEntry(dn DN, entry Entry) error {
	pDn := dn.GetParentDN()
	pNode, err := d.getNode(pDn)
	if err != nil {
		return err
	}

	// put the rdn for the new entry in the entry attrs
	rdn := dn.rdns[len(dn.rdns)-1]
	for ava := range rdn.avas {
		entry.AddAttr(ava)
	}

	entry.dn = dn.Clone()
	pNode.AddChild(entry)
	return nil
}

func (d DIT) ModifyEntry(dn DN, ops ...ChangeOperation) error {
	node, err := d.getNode(dn)
	if err != nil {
		return err
	}

	// by cloning the entry and assigning the new entry to the node, this operation becomes atomic
	entry := node.entry.Clone()
	for _, op := range ops {
		if err = op(&entry); err != nil {
			return err
		}
	}

	node.entry = entry
	return nil
}

func (d DIT) ModifyEntryDN(dn DN, rdn RDN, deleteOldRDN bool, newSuperiorDN *DN) error {
	curr, err := d.getNode(dn)
	if err != nil {
		return err
	}

	curr.entry.SetRDN(rdn, deleteOldRDN)

	// nothing else todo if not moving the node
	if newSuperiorDN == nil {
		return nil
	}

	newParent, err := d.getNode(*newSuperiorDN)
	if err != nil {
		return err // TODO do i need to wrap this so i know it's a different notfound/nosuchobject?
	}

	currParent := curr.parent
	newDn := newSuperiorDN.Clone()
	newDn.AddRDN(rdn)
	entry := curr.entry
	entry.dn = newDn

	currParent.DeleteChild(curr)
	newParent.AddChild(entry)

	return nil
}

func (d DIT) DeleteEntry(dn DN) error {
	node, err := d.getNode(dn)
	if err != nil {
		return err
	}

	if len(node.children) > 0 {
		return ErrNodeNotLeaf
	}

	p := node.parent
	delete(p.children, node)

	return nil
}

func (d DIT) ContainsAttribute(dn DN, ava AVA) (bool, error) {
	node, err := d.getNode(dn)
	if err != nil {
		return false, err
	}

	return node.entry.ContainsAttr(ava), nil
}

func (d DIT) getNode(dn DN) (*DITNode, error) {
	node, err := getNodeRecursive(dn.rdns, d.root)

	var nfErr *NodeNotFoundError
	if errors.As(err, &nfErr) {
		nfErr.requestedDN = dn
		return nil, nfErr
	} else if err != nil {
		return nil, err
	}

	return node, nil
}

func getNodeRecursive(rdns []RDN, node *DITNode) (*DITNode, error) {
	if !node.entry.MatchesRdn(rdns[0]) {
		log.Printf("rdn %s did not match entry", rdns[0])
		return nil, &NodeNotFoundError{}
	}

	if len(rdns) == 1 {
		return node, nil
	}

	var finalErr error
	var nfErr *NodeNotFoundError

	for c := range node.children {
		n, err := getNodeRecursive(rdns[1:], c)

		if err == nil {
			return n, nil
		}

		if !errors.As(err, &nfErr) {
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

type WalkTreeFunc func(Entry)

func WalkTree(n *DITNode, fn WalkTreeFunc) {
	fn(n.entry)
	for c := range n.children {
		WalkTree(n, fn)
	}
}
