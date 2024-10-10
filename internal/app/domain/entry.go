package domain

type Entry struct {
	Id, Parent ID
	Children   map[ID]bool
	ObjClasses map[OID]bool
	Attrs      map[OID]map[string]bool
}
