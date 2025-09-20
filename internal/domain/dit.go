package model

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/georgib0y/relientldap/internal/util"
)

var logger = log.New(os.Stderr, "model: ", log.Lshortfile)

type DITNode struct {
	parent   *DITNode
	children map[*DITNode]struct{}
	entry    *Entry
}

func NewDITNode(parent *DITNode, entry *Entry) *DITNode {
	return &DITNode{
		parent:   parent,
		children: map[*DITNode]struct{}{},
		entry:    entry,
	}
}

func (n *DITNode) AddChildNode(node *DITNode) {
	n.children[node] = struct{}{}
}

func (n *DITNode) AddChild(entry *Entry) {
	c := NewDITNode(n, entry)
	n.children[c] = struct{}{}
}

func (n *DITNode) DeleteChild(child *DITNode) {
	delete(n.children, child)
}

// TODO domain context
type DIT struct {
	root *DITNode
}

func NewDIT(root *DITNode) *DIT {
	return &DIT{root}
}

func (d *DIT) GetEntry(dn DN) (*Entry, error) {
	logger.Printf("getting entry: %s", dn)
	node, err := d.getNode(dn)
	if err != nil {
		return nil, err
	}

	return node.entry, nil
}

func (d *DIT) InsertEntry(dn DN, entry *Entry) error {
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

func (d *DIT) ModifyEntry(dn DN, ops ...ChangeOperation) error {
	node, err := d.getNode(dn)
	if err != nil {
		return err
	}

	// by cloning the entry and assigning the new entry to the node, this operation becomes atomic
	entry := node.entry.Clone()
	for _, op := range ops {
		if err = op(entry); err != nil {
			return err
		}
	}

	node.entry = entry
	return nil
}

func (d *DIT) ModifyEntryDN(dn DN, rdn RDN, deleteOldRDN bool, newSuperiorDN *DN) error {
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
		logger.Printf("could not find parent node at: %s", newSuperiorDN)
		// TODO do i need to wrap this so i know it's a different notfound/nosuchobject?
		return err
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

func (d *DIT) DeleteEntry(dn DN) error {
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

func (d *DIT) ContainsAttribute(dn DN, attr *Attribute, val string) (bool, error) {
	node, err := d.getNode(dn)
	if err != nil {
		return false, err
	}

	return node.entry.ContainsAttrVal(attr, val)
}

func (d *DIT) getNode(dn DN) (*DITNode, error) {
	node, err := getNodeRecursive(dn.rdns, d.root)

	var nfErr *NodeNotFoundError
	if errors.As(err, &nfErr) {
		nfErr.RequestedDN = dn
		return nil, NewLdapError(NoSuchObject, nfErr.MatchedDN.String(), "no object found for requested dn %s", nfErr.RequestedDN)
	} else if err != nil {
		return nil, err
	}

	return node, nil
}

func getNodeRecursive(rdns []RDN, node *DITNode) (*DITNode, error) {
	logger.Printf("getting node at: %s", rdns[0])
	matches, err := node.entry.MatchesRdn(rdns[0])
	if err != nil {
		return nil, err
	}

	if !matches {
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

type WalkTreeFunc func(*Entry)

func WalkTree(n *DITNode, fn WalkTreeFunc) {
	fn(n.entry)
	for c := range n.children {
		WalkTree(c, fn)
	}
}

func WriteNodeDescendants(w io.Writer, node *DITNode) {
	w.Write([]byte("\n"))
	writeNodeRec(w, node, 0)
}

func writeNodeRec(w io.Writer, node *DITNode, indent int) {
	var sb strings.Builder

	for _ = range indent {
		sb.WriteRune('-')
	}
	sb.WriteString("| ")
	sb.WriteString(node.entry.dn.String())
	fmt.Fprintf(&sb, " %v\n", node.entry)

	fmt.Fprintln(w, sb.String())

	for c := range node.children {
		writeNodeRec(w, c, indent+2)
	}
}

/*
Test DIT Structue
| dc=dev
--| dc=georgiboy
----| cn=Test1
----| ou=TestOu
------| cn=Test2
------| cn=Test3
*/
func GenerateTestDIT(schema *Schema) DIT {
	attrs := map[string]*Attribute{}

	attrNames := []string{"dc", "ou", "cn", "sn", "userPassword"}
	for _, name := range attrNames {
		a, ok := schema.FindAttribute(name)
		if !ok {
			logger.Panicf("could not find attribute %q in schema", name)
		}
		attrs[name] = a
	}

	objClasses := map[string]*ObjectClass{}

	ocNames := []string{"dcObject", "person", "organizationalUnit"}
	for _, name := range ocNames {
		o, ok := schema.FindObjectClass(name)
		if !ok {
			logger.Panicf("could not find object class %q in schema", name)
		}
		objClasses[name] = o
	}

	dcDevDn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev").
		Build()
	dcDev := NewDITNode(nil, util.Unwrap(NewEntry(schema,
		dcDevDn,
		WithStructural(objClasses["dcObject"]), // FAIL namingcontexts
		WithEntryAttr(attrs["dc"], "dev"),
	)))

	dcGeorgiboyDn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		Build()
	dcGeorgiboy := NewDITNode(dcDev, util.Unwrap(NewEntry(schema, dcGeorgiboyDn,
		WithStructural(objClasses["dcObject"]),
		WithEntryAttr(attrs["dc"], "georgiboy"),
	)))

	ouTestOuDn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		AddAvaAsRdn(attrs["ou"], "TestOu").
		Build()
	ouTestOu := NewDITNode(dcGeorgiboy, util.Unwrap(NewEntry(schema, ouTestOuDn,
		WithStructural(objClasses["organizationalUnit"]),
		WithEntryAttr(attrs["ou"], "TestOu"))))

	cnTest1Dn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		AddAvaAsRdn(attrs["cn"], "Test1").
		Build()
	cnTest1 := NewDITNode(dcGeorgiboy, util.Unwrap(NewEntry(schema, cnTest1Dn,
		WithStructural(objClasses["person"]),
		WithEntryAttr(attrs["cn"], "Test1"),
		WithEntryAttr(attrs["sn"], "One"),
		WithEntryAttr(attrs["sn"], "Tester"),
		WithEntryAttr(attrs["userPassword"], "password123"),
	)))

	cnTest2Dn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		AddAvaAsRdn(attrs["ou"], "TestOu").AddAvaAsRdn(attrs["cn"], "Test2").
		Build()
	cnTest2 := NewDITNode(ouTestOu, util.Unwrap(NewEntry(schema, cnTest2Dn,
		WithStructural(objClasses["person"]),
		WithEntryAttr(attrs["cn"], "Test2"),
		WithEntryAttr(attrs["sn"], "Tester"),
	)))

	cnTest3Dn := NewDnBuilder().
		AddNamingContext(attrs["dc"], "dev", "georgiboy").
		AddAvaAsRdn(attrs["ou"], "TestOu").AddAvaAsRdn(attrs["cn"], "Test3").
		Build()
	cnTest3 := NewDITNode(ouTestOu, util.Unwrap(NewEntry(schema, cnTest3Dn,
		WithStructural(objClasses["person"]),
		WithEntryAttr(attrs["cn"], "Test3"),
		WithEntryAttr(attrs["sn"], "Tester"),
	)))

	// add each child entry to their parent node
	ouTestOu.AddChildNode(cnTest2)
	ouTestOu.AddChildNode(cnTest3)
	dcGeorgiboy.AddChildNode(cnTest1)
	dcGeorgiboy.AddChildNode(ouTestOu)
	dcDev.AddChildNode(dcGeorgiboy)

	return DIT{dcDev}
}
