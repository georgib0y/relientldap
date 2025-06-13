package ldif

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	m "github.com/georgib0y/relientldap/internal/model"
)

var logger = log.New(os.Stderr, "parser: ", log.Lshortfile)

type schemaBuilder[T m.SchemaObject] interface {
	Build() T
}

type builderResolver[T m.SchemaObject] interface {
	schemaBuilder[T]
	Resolve(oids map[m.OID]T) error
}

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

func (p *tokenParser) parseNumericoid() (m.OID, error) {
	t, ok := p.NextToken()
	if !ok {
		return m.OID(""), fmt.Errorf("expected numericoid got eof")
	}

	if t.tokenType != NUMERICOID {
		return m.OID(""), fmt.Errorf("expected numericoid, got %s: %s", t.tokenType, t.val)
	}

	p.state = EXPECT_KEYWORD
	return m.OID(t.val), nil
}

func (p *tokenParser) parseNoidlen() (m.OID, int, error) {
	t, ok := p.NextToken()
	if !ok {
		return m.OID(""), 0, fmt.Errorf("expected numericoid/noidlen got eof")
	}

	if t.tokenType == NUMERICOID {
		p.state = EXPECT_KEYWORD
		return m.OID(t.val), 0, nil
	}

	if t.tokenType != NOIDLEN {
		return m.OID(""), 0, fmt.Errorf("expected numericoid or noidlen, got %s: %s", t.tokenType, t.val)
	}

	// at this point t.val is definitly in the correct noidlen format
	spl := strings.Split(t.val, "{")
	oid := m.OID(spl[0])
	len, err := strconv.Atoi(spl[1][:len(spl[1])-1])
	if err != nil {
		return m.OID(""), 0, fmt.Errorf("Failed to parse noidlen len (very unexpectedly!): %w", err)
	}

	p.state = EXPECT_KEYWORD
	return oid, len, nil
}

func (p *tokenParser) parseDescr() (string, error) {
	t, ok := p.NextToken()
	if !ok {
		return "", fmt.Errorf("parsing descr got eof")
	}

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

func (p *tokenParser) parseOid() (m.OID, error) {
	t, ok := p.NextToken()
	if !ok {
		return m.OID(""), fmt.Errorf("parsing oid got eof")
	}

	if t.tokenType != NUMERICOID && t.tokenType != KEYWORD && t.tokenType != DESCR {
		return m.OID(""), fmt.Errorf("expected numericoid, keyword or descr, got %s: %s", t.tokenType, t.val)
	}

	p.state = EXPECT_KEYWORD
	return m.OID(t.val), nil
}

func (p *tokenParser) parseOids() ([]m.OID, error) {
	t, ok := p.NextToken()
	if !ok {
		return []m.OID{}, fmt.Errorf("parsing oid got eof")
	}

	// if only one oid
	// keyword is a subset of descr and will be matched before it first
	if t.tokenType == NUMERICOID || t.tokenType == KEYWORD || t.tokenType == DESCR {
		p.state = EXPECT_KEYWORD
		return []m.OID{m.OID(t.val)}, nil
	}

	if t.tokenType != LPAREN {
		return []m.OID{}, fmt.Errorf("expected oid or lparen, got %s: %s", t.tokenType, t.val)
	}

	oids := []m.OID{}
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

		oids = append(oids, m.OID(t.val))
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

func resolveDepends[T m.SchemaObject](builders map[builderResolver[T]]struct{}) (map[m.OID]T, error) {
	resolved := map[m.OID]T{}
	last := 0
	builders_count := len(builders)

outter:
	for len(resolved) < builders_count {
		for b := range builders {
			if err := b.Resolve(resolved); err != nil {
				continue
			}

			r := b.Build()
			resolved[r.Oid()] = r
			delete(builders, b)
		}

		if len(resolved) == last {
			break outter
		}
		last = len(resolved)
	}

	if len(resolved) < builders_count {
		return nil, fmt.Errorf("Unable to resolve all superior dependencies (%d/%d unresolved)", len(resolved), builders_count)
	}

	return resolved, nil
}

type Parser[T m.SchemaObject] interface {
	SetTokens(tokens []Token)
	NextToken() (Token, bool)
	Builder() schemaBuilder[T]
	HandleNumericoid(oid m.OID) error
	HandleKeyword(kw string) error
}

func Parse[T m.SchemaObject](p Parser[T]) error {
	// numericoid is always the first element in the m.ma
	t, ok := p.NextToken()
	if !ok {
		return fmt.Errorf("End of tokens, expected numericoid")
	}
	if t.tokenType != NUMERICOID {
		return fmt.Errorf("expected numericoid first, got %s: %s", t.tokenType, t.val)
	}

	if err := p.HandleNumericoid(m.OID(t.val)); err != nil {
		return err
	}

	for {
		t, ok := p.NextToken()
		if !ok {
			break
		}

		if t.tokenType != KEYWORD {
			return fmt.Errorf("expected keyword, got %s: %s", t.tokenType, t.val)
		}

		if err := p.HandleKeyword(t.val); err != nil {
			return err
		}
	}

	return nil
}

func ParseReader[T m.SchemaObject](r io.Reader, p Parser[T]) (map[m.OID]T, error) {
	tokens, err := tokenise(r)
	if err != nil {
		return nil, err
	}

	if !checkTokensBalanced(tokens) {
		return nil, fmt.Errorf("Tokens unbalanced")
	}

	builders := []schemaBuilder[T]{}

	idx := 0
	for idx < len(tokens) {
		end, err := findMatchingParen(tokens, idx)
		if err != nil {
			return nil, err
		}

		defTokens := tokens[idx+1 : end-1]
		p.SetTokens(defTokens)

		if err = Parse(p); err != nil {
			return nil, err
		}

		builders = append(builders, p.Builder())
		idx = end
	}

	if _, ok := p.Builder().(builderResolver[T]); ok {
		brs := map[builderResolver[T]]struct{}{}
		for _, b := range builders {
			br := b.(builderResolver[T])
			brs[br] = struct{}{}
		}

		return resolveDepends(brs)
	}

	objects := map[m.OID]T{}
	for _, b := range builders {
		o := b.Build()
		objects[o.Oid()] = o
	}

	return objects, nil
}

type ObjectClassParser struct {
	tokenParser
	attrs map[m.OID]*m.Attribute
	b     *m.ObjectClassBuilder
}

func NewObjectClassParser(attrs map[m.OID]*m.Attribute) *ObjectClassParser {
	return &ObjectClassParser{
		tokenParser: tokenParser{
			tokens: []Token{},
			idx:    0,
			state:  UNINITIALISED,
		},
		attrs: attrs,
		b:     m.NewObjectClassBuilder(),
	}
}

func (o *ObjectClassParser) SetTokens(tokens []Token) {
	o.setTokens(tokens)
	o.b = m.NewObjectClassBuilder()
}

func (o *ObjectClassParser) Builder() schemaBuilder[*m.ObjectClass] {
	return o.b
}

func (o *ObjectClassParser) HandleNumericoid(oid m.OID) error {
	o.b.SetOid(oid)
	return nil
}

func (o *ObjectClassParser) HandleKeyword(kw string) error {
	switch kw {
	case "NAME":
		names, err := o.parseQdescr()
		if err != nil {
			return err
		}
		o.b.AddName(names...)
	case "DESC":
		desc, err := o.parseQdstring()
		if err != nil {
			return err
		}
		o.b.SetDesc(desc)
	case "OBSOLETE":
		o.b.SetObsolete()
	case "SUP":
		oids, err := o.parseOids()
		if err != nil {
			return err
		}
		o.b.AddSupName(oids...)
	case "ABSTRACT":
		o.b.SetKind(m.Abstract)
	case "STRUCTURAL":
		o.b.SetKind(m.Structural)
	case "AUXILIARY":
		o.b.SetKind(m.Auxilary)
	case "MUST":
		oids, err := o.parseOids()
		if err != nil {
			return err
		}

		for _, oid := range oids {
			attr, ok := o.attrs[oid]
			if !ok {
				return fmt.Errorf("Unknown attribute id %s", oid)
			}
			o.b.AddMustAttr(attr)
		}
	case "MAY":
		oids, err := o.parseOids()
		if err != nil {
			return err
		}
		for _, oid := range oids {
			attr, ok := o.attrs[oid]
			if !ok {
				return fmt.Errorf("Unknown attribute id %s", oid)
			}
			o.b.AddMayAttr(attr)
		}
	default:
		return fmt.Errorf("Unknown Object Class keyword: '%s'", kw)
	}

	return nil
}

type AttributeParser struct {
	tokenParser
	b *m.AttributeBuilder
}

func NewAttributeParser() *AttributeParser {
	return &AttributeParser{
		tokenParser: tokenParser{
			tokens: []Token{},
			idx:    0,
			state:  UNINITIALISED,
		},
		b: m.NewAttributeBuilder(),
	}
}

func (a *AttributeParser) SetTokens(tokens []Token) {
	a.setTokens(tokens)
	a.b = m.NewAttributeBuilder()
}

func (a *AttributeParser) Builder() schemaBuilder[*m.Attribute] {
	return a.b
}

func (a *AttributeParser) HandleNumericoid(oid m.OID) error {
	a.b.SetOid(oid)
	return nil
}

func (a *AttributeParser) HandleKeyword(kw string) error {
	switch kw {
	case "NAME":
		names, err := a.parseQdescr()
		if err != nil {
			return err
		}
		a.b.AddNames(names...)
	case "DESC":
		desc, err := a.parseQdstring()
		if err != nil {
			return err
		}
		a.b.SetDesc(desc)
	case "OBSOLETE":
		a.b.SetObsolete()
	case "SUP":
		sup, err := a.parseOid()
		if err != nil {
			return err
		}
		a.b.SetSupOid(sup)
	case "EQUALITY":
		eq, err := a.parseOid()
		if err != nil {
			return err
		}

		rule, ok := m.GetMatchingRule(string(eq))
		if !ok {
			return fmt.Errorf("Unknown equality rule %s", eq)
		}
		a.b.SetEqRule(rule)
	case "ORDERING":
		ord, err := a.parseOid()
		if err != nil {
			return err
		}

		rule, ok := m.GetMatchingRule(string(ord))
		if !ok {
			return fmt.Errorf("Unknown ordering rule %s", ord)
		}
		a.b.SetOrdRule(rule)
	case "SUBSTR":
		sub, err := a.parseOid()
		if err != nil {
			return err
		}

		rule, ok := m.GetMatchingRule(string(sub))
		if !ok {
			return fmt.Errorf("Unknown ordering rule %s", sub)
		}
		a.b.SetSubStrRule(rule)
	case "SYNTAX":
		syntax, len, err := a.parseNoidlen()
		if err != nil {
			return err
		}
		a.b.SetSyntax(syntax, len)
	case "SINGLE-VALUE":
		a.b.SetSingleVal()
	case "COLLECTIVE":
		a.b.SetCollective()
	case "NO-USER-MODIFICATION":
		a.b.SetNoUserMod()
	case "USAGE":
		u, err := a.parseDescr()
		if err != nil {
			return err
		}
		usage, err := m.NewUsage(u)
		if err != nil {
			return err
		}
		a.b.SetUsage(usage)
	default:
		return fmt.Errorf("Unknown Attribute keyword: '%s'", kw)
	}

	return nil
}
