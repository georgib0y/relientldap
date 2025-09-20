package domain

type Syntax struct {
	numericoid OID
	desc       string
	validate   func(s string) error
}

func (s Syntax) Validate(v string) error {
	if s.validate == nil {
		return NewLdapError(UnwillingToPerform, nil, "Syntax %q is unimplemented", s.numericoid)
	}
	return s.validate(v)
}

func (s Syntax) Eq(o Syntax) bool {
	return s.numericoid == o.numericoid
}

var syntaxes = map[string]Syntax{
	"1.3.6.1.4.1.1466.115.121.1.3": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.3",
		desc:       "Attribute Type Description",
	},
	"1.3.6.1.4.1.1466.115.121.1.6": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.6",
		desc:       "Bit String",
	},
	"1.3.6.1.4.1.1466.115.121.1.7": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.7",
		desc:       "Boolean",
		validate:   validateBoolean,
	},
	"1.3.6.1.4.1.1466.115.121.1.11": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.11",
		desc:       "Country String",
	},
	"1.3.6.1.4.1.1466.115.121.1.14": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.14",
		desc:       "Delivery Method",
	},
	"1.3.6.1.4.1.1466.115.121.1.15": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.15",
		desc:       "Directory String",
		validate:   validateDirectoryString,
	},
	"1.3.6.1.4.1.1466.115.121.1.16": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.16",
		desc:       "DIT Content Rule Description",
	},
	"1.3.6.1.4.1.1466.115.121.1.17": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.17",
		desc:       "DIT Structure Rule Description",
	},
	"1.3.6.1.4.1.1466.115.121.1.12": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.12",
		desc:       "DN",
	},
	"1.3.6.1.4.1.1466.115.121.1.21": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.21",
		desc:       "Enhanced Guide",
	},
	"1.3.6.1.4.1.1466.115.121.1.22": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.22",
		desc:       "Facsimile Telephone Number",
	},
	"1.3.6.1.4.1.1466.115.121.1.23": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.23",
		desc:       "Fax",
	},
	"1.3.6.1.4.1.1466.115.121.1.24": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.24",
		desc:       "Generalized Time",
	},
	"1.3.6.1.4.1.1466.115.121.1.25": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.25",
		desc:       "Guide",
	},
	"1.3.6.1.4.1.1466.115.121.1.26": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.26",
		desc:       "IA5 String",
		validate:   validateIA5String,
	},
	"1.3.6.1.4.1.1466.115.121.1.27": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.27",
		desc:       "INTEGER",
	},
	"1.3.6.1.4.1.1466.115.121.1.28": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.28",
		desc:       "JPEG",
	},
	"1.3.6.1.4.1.1466.115.121.1.54": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.54",
		desc:       "LDAP Syntax Description",
	},
	"1.3.6.1.4.1.1466.115.121.1.30": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.30",
		desc:       "Matching Rule Description",
	},
	"1.3.6.1.4.1.1466.115.121.1.31": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.31",
		desc:       "Matching Rule Use Description",
	},
	"1.3.6.1.4.1.1466.115.121.1.34": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.34",
		desc:       "Name And Optional UID",
	},
	"1.3.6.1.4.1.1466.115.121.1.35": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.35",
		desc:       "Name Form Description",
	},
	"1.3.6.1.4.1.1466.115.121.1.36": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.36",
		desc:       "Numeric String",
	},
	"1.3.6.1.4.1.1466.115.121.1.37": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.37",
		desc:       "Object Class Description",
	},
	"1.3.6.1.4.1.1466.115.121.1.40": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.40",
		desc:       "Octet String",
		validate:   validateOctetString,
	},
	"1.3.6.1.4.1.1466.115.121.1.38": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.38",
		desc:       "OID",
	},
	"1.3.6.1.4.1.1466.115.121.1.39": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.39",
		desc:       "Other Mailbox",
	},
	"1.3.6.1.4.1.1466.115.121.1.41": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.41",
		desc:       "Postal Address",
	},
	"1.3.6.1.4.1.1466.115.121.1.44": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.44",
		desc:       "Printable String",
	},
	"1.3.6.1.4.1.1466.115.121.1.58": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.58",
		desc:       "Substring Assertion",
	},
	"1.3.6.1.4.1.1466.115.121.1.50": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.50",
		desc:       "Telephone Number",
	},
	"1.3.6.1.4.1.1466.115.121.1.51": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.51",
		desc:       "Teletex Terminal Identifier",
	},
	"1.3.6.1.4.1.1466.115.121.1.52": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.52",
		desc:       "Telex Number",
	},
	"1.3.6.1.4.1.1466.115.121.1.53": Syntax{
		numericoid: "1.3.6.1.4.1.1466.115.121.1.53",
		desc:       "UTC Time",
	},
}

func GetSyntax(oid OID) (Syntax, error) {
	s, ok := syntaxes[string(oid)]
	if !ok {
		return Syntax{}, NewLdapError(InvalidAttributeSyntax, nil, "Unknown syntax %q", oid)
	}
	return s, nil
}

func validateBoolean(s string) error {
	switch s {
	case "TRUE":
		return nil
	case "FALSE":
		return nil
	default:
		return NewLdapError(InvalidAttributeSyntax, nil, "unknown boolean %q", s)
	}
}

func validateIA5String(s string) error {
	// ia5 strings are a bit like ascii but in different order,
	// technically could be invalid (highest bit set is invalid)
	return nil
}

func validateDirectoryString(s string) error {
	if len(s) == 0 {
		return NewLdapError(InvalidAttributeSyntax, nil, "directory strings cannot be empty")
	}

	return nil
}

func validateOctetString(s string) error {
	return nil //anything goes
}
