package ldif

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"

	"github.com/georgib0y/relientldap/internal/model/dit"
	"github.com/georgib0y/relientldap/internal/model/schema"
)

type parserState int

const (
	UNINITIALISED parserState = iota
	EXPECT_NUMERICOID
	EXPECT_KEYWORD
	EXPECT_QDESCRS
	EXPECT_QDSTRING
	EXPECT_OIDS
	// TODO EXPECT_EXTENSIONS
)

func (p parserState) String() string {
	switch p {
	case EXPECT_NUMERICOID:
		return "EXPECT_NUMERICOID"
	case EXPECT_KEYWORD:
		return "EXPECT_KEYWORD"
	case EXPECT_QDESCRS:
		return "EXPECT_QDESCRS"
	case EXPECT_QDSTRING:
		return "EXPECT_QDSTRING"
	case EXPECT_OIDS:
		return "EXPECT_OIDS"
	default:
		return fmt.Sprintf("unknown state %d", p)
	}
}

var ocKwExpect = map[string]parserState{
	"NAME":       EXPECT_QDESCRS,
	"DESC":       EXPECT_QDSTRING,
	"OBSOLETE":   EXPECT_KEYWORD,
	"SUP":        EXPECT_OIDS,
	"ABSTRACT":   EXPECT_KEYWORD,
	"STRUCTURAL": EXPECT_KEYWORD,
	"AUXILIARY":  EXPECT_KEYWORD,
	"MUST":       EXPECT_OIDS,
	"MAY":        EXPECT_OIDS,
}

type tokenParser struct {
	tokens []Token
	idx    int
	state  parserState
}

func (p *tokenParser) setTokens(t []Token) {
	p.tokens = t
	p.idx = 0
	p.state = EXPECT_NUMERICOID
}

func (p *tokenParser) NextToken() (Token, bool) {
	if p.idx == len(p.tokens) {
		return Token{}, false
	}

	t := p.tokens[p.idx]
	p.idx++
	return t, true
}

func (p *tokenParser) parseNumericoid() (dit.OID, error) {
	t, ok := p.NextToken()
	if !ok {
		return dit.OID(""), fmt.Errorf("expected numericoid got eof")
	}

	log.Printf("token is %s: %s", t.tokenType, t.val)

	if t.tokenType != NUMERICOID {
		return dit.OID(""), fmt.Errorf("expected numericoid, got %s: %s", t.tokenType, t.val)
	}

	p.state = EXPECT_KEYWORD
	return dit.OID(t.val), nil
}

func (p *tokenParser) parseNoidlen() (dit.OID, int, error) {
	t, ok := p.NextToken()
	if !ok {
		return dit.OID(""), 0, fmt.Errorf("expected numericoid/noidlen got eof")
	}

	log.Printf("token is %s: %s", t.tokenType, t.val)

	if t.tokenType == NUMERICOID {
		p.state = EXPECT_KEYWORD
		return dit.OID(t.val), 0, nil
	}

	if t.tokenType != NOIDLEN {
		return dit.OID(""), 0, fmt.Errorf("expected numericoid or noidlen, got %s: %s", t.tokenType, t.val)
	}

	// at this point t.val is definitly in the correct noidlen format
	spl := strings.Split(t.val, "{")
	oid := dit.OID(spl[0])
	len, err := strconv.Atoi(spl[1][:len(spl[1])-1])
	if err != nil {
		return dit.OID(""), 0, fmt.Errorf("Failed to parse noidlen len (very unexpectedly!): %w", err)
	}

	p.state = EXPECT_KEYWORD
	return oid, len, nil
}

func (p *tokenParser) parseDescr() (string, error) {
	t, ok := p.NextToken()
	if !ok {
		return "", fmt.Errorf("parsing descr got eof")
	}

	log.Printf("token is %s: %s", t.tokenType, t.val)

	if t.tokenType != DESCR {
		return "", fmt.Errorf("Failed to parse descr, got %s: %s", t.tokenType, t.val)
	}

	return t.val, nil
}

func (p *tokenParser) parseQdescr() ([]string, error) {
	t, ok := p.NextToken()
	if !ok {
		return nil, fmt.Errorf("parsing qdescr got eof")
	}

	log.Printf("token is %s: %s", t.tokenType, t.val)
	log.Printf("state is %s", p.state)

	// if only one qdescr
	if t.tokenType == QDESCR {
		p.state = EXPECT_KEYWORD
		return []string{stripQuotes(t.val)}, nil
	}

	if t.tokenType != LPAREN {
		return nil, fmt.Errorf("expected qdescr or lparen, got %s: %s", t.tokenType, t.val)
	}

	qdescrs := []string{}
	for {
		next, ok := p.NextToken()
		if !ok {
			return nil, fmt.Errorf("parsing qdescrs got eof")
		}

		// TODO is this needed for shadowing?
		t = next

		if t.tokenType == RPAREN {
			p.state = EXPECT_KEYWORD
			return qdescrs, nil
		}

		if t.tokenType != QDESCR {
			return nil, fmt.Errorf("expected qdescr or rparen, got %s: %s", t.tokenType, t.val)
		}

		qdescrs = append(qdescrs, stripQuotes(t.val))
	}
}

func (p *tokenParser) parseQdstring() (string, error) {
	t, ok := p.NextToken()
	if !ok {
		return "", fmt.Errorf("parsing qdstring got eof")
	}

	// qdescr is a subset of qdstring, the tokeniser will match qdescr first - so allow both
	if t.tokenType != QDESCR && t.tokenType != QDSTRING {
		return "", fmt.Errorf("expected qdescr or qdstring, got %s: %s", t.tokenType, t.val)
	}

	p.state = EXPECT_KEYWORD
	return stripQuotes(t.val), nil
}

func (p *tokenParser) parseOid() (dit.OID, error) {
	t, ok := p.NextToken()
	if !ok {
		return dit.OID(""), fmt.Errorf("parsing oid got eof")
	}

	if t.tokenType != NUMERICOID && t.tokenType != KEYWORD && t.tokenType != DESCR {
		return dit.OID(""), fmt.Errorf("expected numericoid, keyword or descr, got %s: %s", t.tokenType, t.val)
	}

	p.state = EXPECT_KEYWORD
	return dit.OID(t.val), nil
}

func (p *tokenParser) parseOids() ([]dit.OID, error) {
	t, ok := p.NextToken()
	if !ok {
		return []dit.OID{}, fmt.Errorf("parsing oid got eof")
	}

	// if only one oid
	// keyword is a subset of descr and will be matched before it first
	if t.tokenType == NUMERICOID || t.tokenType == KEYWORD || t.tokenType == DESCR {
		p.state = EXPECT_KEYWORD
		return []dit.OID{dit.OID(t.val)}, nil
	}

	if t.tokenType != LPAREN {
		return []dit.OID{}, fmt.Errorf("expected oid or lparen, got %s: %s", t.tokenType, t.val)
	}

	oids := []dit.OID{}
	for {
		next, ok := p.NextToken()
		if !ok {
			return oids, fmt.Errorf("parsing oids got eof")
		}

		// skip $ separators
		if next.tokenType == DOLLAR {
			if next, ok = p.NextToken(); !ok {
				return oids, fmt.Errorf("parsing oids got eof")
			}
		}

		// TODO is this needed for shadowing?
		t = next

		if t.tokenType == RPAREN {
			p.state = EXPECT_KEYWORD
			return oids, nil
		}

		if t.tokenType != DESCR && t.tokenType != KEYWORD {
			return oids, fmt.Errorf("expected descr or rparen, got %s: %s", t.tokenType, t.val)
		}

		oids = append(oids, dit.OID(t.val))
	}
}

func findMatchingParen(tokens []Token, lparenIdx int) (int, error) {
	if tokens[lparenIdx].tokenType != LPAREN {
		return 0, fmt.Errorf("Expected LPAREN, got %s: %s",
			tokens[lparenIdx].tokenType,
			tokens[lparenIdx].val,
		)
	}

	bal := 1
	rparenIdx := lparenIdx + 1
	for ; rparenIdx < len(tokens) && bal != 0; rparenIdx += 1 {
		if tokens[rparenIdx].tokenType == LPAREN {
			bal += 1
		}

		if tokens[rparenIdx].tokenType == RPAREN {
			bal -= 1
		}
	}

	if bal != 0 {
		return 0, fmt.Errorf("Could not find matching rparen")
	}

	return rparenIdx, nil
}

type Parser[T any] interface {
	SetTokens([]Token)
	NextToken() (Token, bool)
	Build() (T, error)
	HandleNumericoid(oid dit.OID) error
	HandleKeyword(kw string) error
}

func Parse[T any](p Parser[T]) (T, error) {
	var zero T
	// numericoid is always the first element in the schema
	t, ok := p.NextToken()
	if !ok {
		return zero, fmt.Errorf("End of tokens, expected numericoid")
	}
	if t.tokenType != NUMERICOID {
		return zero, fmt.Errorf("expected numericoid first, got %s: %s", t.tokenType, t.val)
	}

	if err := p.HandleNumericoid(dit.OID(t.val)); err != nil {
		return zero, err
	}

	for {
		t, ok := p.NextToken()
		if !ok {
			break
		}
		if t.tokenType != KEYWORD {
			return zero, fmt.Errorf("expected keyword, got %s: %s", t.tokenType, t.val)
		}

		if err := p.HandleKeyword(t.val); err != nil {
			return zero, err
		}
	}

	return p.Build()
}

func ParseReader[T any](r io.Reader, p Parser[T]) ([]T, error) {
	tokens, err := tokenise(r)
	if err != nil {
		return nil, err
	}

	if !checkTokensBalanced(tokens) {
		return nil, fmt.Errorf("Tokens unbalanced")
	}

	defs := []T{}

	idx := 0
	for idx < len(tokens) {
		end, err := findMatchingParen(tokens, idx)
		if err != nil {
			return nil, err
		}

		defTokens := tokens[idx+1 : end-1]
		log.Print("---- tokens ----")
		for _, t := range defTokens {
			log.Printf("%s: %s", t.tokenType, t.val)
		}
		log.Print("---- end tokens ----")
		p.SetTokens(defTokens)

		oc, err := Parse(p)
		if err != nil {
			return nil, err
		}
		defs = append(defs, oc)
		idx = end
	}

	return defs, nil
}

type ObjectClassParser struct {
	tokenParser
	opts []schema.ObjectClassOption
}

func NewObjectClassParser() *ObjectClassParser {
	return &ObjectClassParser{
		tokenParser: tokenParser{
			tokens: []Token{},
			idx:    0,
			state:  UNINITIALISED,
		},
		opts: []schema.ObjectClassOption{},
	}
}

func (o *ObjectClassParser) SetTokens(t []Token) {
	o.setTokens(t)
	o.opts = []schema.ObjectClassOption{}
}

func (o *ObjectClassParser) addOpt(opt schema.ObjectClassOption) {
	o.opts = append(o.opts, opt)
}

func (o *ObjectClassParser) Build() (schema.ObjectClass, error) {
	return schema.NewObjectClass(o.opts...), nil
}

func (o *ObjectClassParser) HandleNumericoid(oid dit.OID) error {
	o.opts = append(o.opts, schema.ObjClassWithOid(oid))
	return nil
}

func (o *ObjectClassParser) HandleKeyword(kw string) error {
	switch kw {
	case "NAME":
		names, err := o.parseQdescr()
		if err != nil {
			return err
		}
		o.addOpt(schema.ObjClassWithName(names...))
	case "DESC":
		desc, err := o.parseQdstring()
		if err != nil {
			return err
		}
		o.addOpt(schema.ObjClassWithDesc(desc))
	case "OBSOLETE":
		o.addOpt(schema.ObjClassWithObsolete())
	case "SUP":
		oids, err := o.parseOids()
		if err != nil {
			return err
		}
		o.addOpt(schema.ObjClassWithSupOid(oids...))
	case "ABSTRACT":
		o.addOpt(schema.ObjClassWithKind(schema.Abstract))
	case "STRUCTURAL":
		o.addOpt(schema.ObjClassWithKind(schema.Structural))
	case "AUXILIARY":
		o.addOpt(schema.ObjClassWithKind(schema.Auxilary))
	case "MUST":
		oids, err := o.parseOids()
		if err != nil {
			return err
		}
		o.addOpt(schema.ObjClassWithMustAttr(oids...))
	case "MAY":
		oids, err := o.parseOids()
		if err != nil {
			return err
		}
		o.addOpt(schema.ObjClassWithMayAttr(oids...))
	default:
		return fmt.Errorf("Unknown Object Class keyword: '%s'", kw)
	}

	return nil
}

type AttributeParser struct {
	tokenParser
	opts []schema.AttrOption
}

func NewAttributeParser() *AttributeParser {
	return &AttributeParser{
		tokenParser: tokenParser{
			tokens: []Token{},
			idx:    0,
			state:  UNINITIALISED,
		},
		opts: []schema.AttrOption{},
	}
}

func (a *AttributeParser) addOpt(opt schema.AttrOption) {
	a.opts = append(a.opts, opt)
}

func (a *AttributeParser) Build() (schema.Attribute, error) {
	return schema.NewAttribute(a.opts...), nil
}

func (a *AttributeParser) HandleNumericoid(oid dit.OID) error {
	a.opts = append(a.opts, schema.AttrWithOid(oid))
	return nil
}

func (a *AttributeParser) HandleKeyword(kw string) error {
	switch kw {
	case "NAME":
		names, err := a.parseQdescr()
		if err != nil {
			return err
		}
		a.addOpt(schema.AttrWithName(names...))
	case "DESC":
		desc, err := a.parseQdstring()
		if err != nil {
			return err
		}
		a.addOpt(schema.AttrWithDesc(desc))
	case "OBSOLETE":
		a.addOpt(schema.AttrWithObsolete())
	case "SUP":
		sup, err := a.parseOid()
		if err != nil {
			return err
		}
		a.addOpt(schema.AttrWithOid(sup))
	case "EQUALITY":
		eq, err := a.parseOid()
		if err != nil {
			return err
		}
		a.addOpt(schema.AttrWithEqRule(eq))
	case "ORDERING":
		ord, err := a.parseOid()
		if err != nil {
			return err
		}
		a.addOpt(schema.AttrWithOrdRule(ord))
	case "SUBSTR":
		sub, err := a.parseOid()
		if err != nil {
			return err
		}
		a.addOpt(schema.AttrWithSubstrRule(sub))
	case "SYNTAX":
		syntax, len, err := a.parseNoidlen()
		if err != nil {
			return err
		}
		a.addOpt(schema.AttrWithSyntax(syntax, len))
	case "SINGLE-VALUE":
		a.addOpt(schema.AttrWithSingleVal())
	case "COLLECTIVE":
		a.addOpt(schema.AttrWithCollective())
	case "NO-USER-MODIFICATION":
		a.addOpt(schema.AttrWithNoUserMod())
	case "USAGE":
		u, err := a.parseDescr()
		if err != nil {
			return err
		}
		usage, err := schema.NewUsage(u)
		if err != nil {
			return err
		}
		a.addOpt(schema.AttrWithUsage(usage))
	default:
		return fmt.Errorf("Unknown Attribute keyword: '%s'", kw)
	}

	return nil
}
