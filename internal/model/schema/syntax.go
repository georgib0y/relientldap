package schema

import (
	"fmt"
)

// TODO do i need other infromation ie description?
// type ldapSyntax struct {
// 	descr    string
// 	validate func(string) bool
// 	// TODO extensions?
// }

func validateBoolean(s string) bool {
	switch s {
	case "TRUE":
		return true
	case "FALSE":
		return true
	default:
		return false
	}
}

func ValidateSyntax(syntax OID, val string) (bool, error) {
	switch string(syntax) {
	case "1.3.6.1.4.1.1466.115.121.1.6":
		return validateBoolean(val), nil
	default:
		return false, fmt.Errorf("unknown syntax with oid: %s", syntax)
	}
}
