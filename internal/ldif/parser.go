package ldif

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"

	"github.com/georgib0y/relientldap/internal/model/dit"
	"github.com/georgib0y/relientldap/internal/model/schema"
)

type TokenType int

const (
	NUMERICOID TokenType = iota
	DESCR
	LPAREN
	RPAREN
	QDESCR
	QDSTRING
	KEYWORD
	DOLLAR
)

func (t TokenType) String() string {
	switch t {
	case NUMERICOID:
		return "NUMERICOID"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case QDESCR:
		return "QDESCR"
	case QDSTRING:
		return "QDSTRING"
	case KEYWORD:
		return "KEYWORD"
	case DOLLAR:
		return "DOLLAR"
	}

	return "unknown"
}

var (
	numericoid_re = regexp.MustCompile(`^[0-9]+(\.[0-9]+)+$`)
	keyword_re    = regexp.MustCompile(`^[A-Z]+$`)
	descr_re      = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9-]*$`)
	qdescr_re     = regexp.MustCompile(`^\'[a-zA-Z][a-zA-Z0-9-]*\'$`)
	qdstring_re   = regexp.MustCompile(`^\'[^\\\']+\'$`)
)

func determineTokenType(val string) (TokenType, error) {
	switch {
	case val == "(":
		return LPAREN, nil
	case val == ")":
		return RPAREN, nil
	case val == "$":
		return DOLLAR, nil
	case numericoid_re.Match([]byte(val)):
		return NUMERICOID, nil
	case keyword_re.Match([]byte(val)):
		return KEYWORD, nil
	case descr_re.Match([]byte(val)):
		return DESCR, nil
	case qdescr_re.Match([]byte(val)):
		return QDESCR, nil
	case qdstring_re.Match([]byte(val)):
		return QDSTRING, nil

	}

	return KEYWORD, fmt.Errorf("Unknown token type for '%s'", val)
}

type Token struct {
	val       string
	tokenType TokenType
}

type tokeniserState int

const (
	Normal tokeniserState = iota
	InQuote
)

func isWsp(r rune) bool {
	switch r {
	case ' ':
		return true
	case '\t':
		return true
	case '\n':
		return true
	case '\r': // ew
		return true
	default:
		return false
	}
}

func readNext(r *bufio.Reader) (string, error) {
	var sb strings.Builder
	state := Normal

	for {
		c, _, err := r.ReadRune()

		if err == io.EOF {
			return sb.String(), io.EOF
		}

		if err != nil {
			return "", err
		}

		switch {
		case state == Normal && isWsp(c) && sb.Len() == 0:
			// skip leading whitespace
			continue
		case state == Normal && isWsp(c):
			// break at normal whitespace
			return sb.String(), nil
		case state == Normal && c == '\'':
			sb.WriteRune(c)
			state = InQuote
		case state == InQuote && c == '\'':
			sb.WriteRune(c)
			return sb.String(), nil
		default:
			sb.WriteRune(c)
		}

	}
}

func tokenise(r io.Reader) ([]Token, error) {
	bufr := bufio.NewReader(r)
	tokens := []Token{}

	for {
		tVal, readErr := readNext(bufr)
		if readErr != nil && readErr != io.EOF {
			return nil, readErr
		}

		// skip if at end and there is nothing to process
		if readErr == io.EOF && tVal == "" {
			break
		}

		log.Printf("next tVal is: '%s'", tVal)

		tType, err := determineTokenType(tVal)
		if err != nil {
			return nil, err
		}

		tokens = append(tokens, Token{tVal, tType})

		if readErr == io.EOF {
			break
		}

	}

	return tokens, nil
}

func checkTokensBalanced(tokens []Token) bool {
	l, r := 0, 0
	for _, t := range tokens {
		if t.tokenType == LPAREN {
			l += 1
		}
		if t.tokenType == RPAREN {
			r += 1
		}
	}

	return l == r
}

// assumes s is a known qdescr or qdstring
func stripQuotes(s string) string {
	return s[1 : len(s)-1]
}

type parserState int

const (
	EXPECT_NUMERICOID parserState = iota
	EXPECT_KEYWORD
	EXPECT_NAME
	EXPECT_DESC
	EXPECT_SUP
	EXPECT_MUST
	EXPECT_MAY

	// TODO EXPECT_EXTENSIONS
)

func (p parserState) String() string {
	switch p {
	case EXPECT_NUMERICOID:
		return "EXPECT_NUMERICOID"
	case EXPECT_KEYWORD:
		return "EXPECT_KEYWORD"
	case EXPECT_NAME:
		return "EXPECT_NAME"
	case EXPECT_DESC:
		return "EXPECT_DESC"
	case EXPECT_SUP:
		return "EXPECT_SUP"
	case EXPECT_MUST:
		return "EXPECT_MUST"
	case EXPECT_MAY:
		return "EXPECT_MAY"
	default:
		return fmt.Sprintf("unknown state %d", p)
	}
}

var ocKwExpect = map[string]parserState{
	"NAME":       EXPECT_NAME,
	"DESC":       EXPECT_DESC,
	"OBSOLETE":   EXPECT_KEYWORD,
	"SUP":        EXPECT_SUP,
	"ABSTRACT":   EXPECT_KEYWORD,
	"STRUCTURAL": EXPECT_KEYWORD,
	"AUXILIARY":  EXPECT_KEYWORD,
	"MUST":       EXPECT_MUST,
	"MAY":        EXPECT_MAY,
}

type objectClassParser struct {
	tokens []Token
	idx    int
	state  parserState
	opts   []schema.ObjectClassOption
}

func newObjectClassParser(tokens []Token) *objectClassParser {
	return &objectClassParser{tokens, 0, EXPECT_NUMERICOID, []schema.ObjectClassOption{}}
}

func (p *objectClassParser) nextToken() (Token, bool) {
	if p.idx == len(p.tokens) {
		return Token{}, false
	}

	t := p.tokens[p.idx]
	p.idx += 1
	return t, true
}

func (p *objectClassParser) parseNumericoid() error {
	t, ok := p.nextToken()
	if !ok {
		return fmt.Errorf("expected numericoid got eof")
	}

	log.Printf("token is %s: %s", t.tokenType, t.val)

	if t.tokenType != NUMERICOID {
		return fmt.Errorf("expected numericoid, got %s: %s", t.tokenType, t.val)
	}
	p.opts = append(p.opts, schema.WithOid(dit.OID(t.val)))
	p.state = EXPECT_KEYWORD
	return nil
}

func (p *objectClassParser) parseKeyword() error {
	t, ok := p.nextToken()
	if !ok {
		return io.EOF
	}

	log.Printf("token is %s: %s", t.tokenType, t.val)

	if t.tokenType != KEYWORD {
		return fmt.Errorf("expected keyword, got %s: %s", t.tokenType, t.val)
	}

	e, ok := ocKwExpect[t.val]
	if !ok {
		return fmt.Errorf("unknown keyword '%s'", t.val)
	}

	switch t.val {
	case "OBSOLETE":
		p.opts = append(p.opts, schema.WithObsolete())
	case "ABSTRACT":
		p.opts = append(p.opts, schema.WithKind(schema.Abstract))
	case "STRUCTURAL":
		p.opts = append(p.opts, schema.WithKind(schema.Structural))
	case "AUXILIARY":
		p.opts = append(p.opts, schema.WithKind(schema.Auxilary))
	}

	p.state = e
	return nil
}

func (p *objectClassParser) parseName() error {
	t, ok := p.nextToken()
	if !ok {
		return fmt.Errorf("parsing name got eof")
	}

	log.Printf("token is %s: %s", t.tokenType, t.val)
	log.Printf("state is %s", p.state)

	// if only one qdescr
	if t.tokenType == QDESCR {
		p.opts = append(p.opts, schema.WithName(stripQuotes(t.val)))
		p.state = EXPECT_KEYWORD
		return nil
	}

	if t.tokenType != LPAREN {
		return fmt.Errorf("expected qdescr or lparen, got %s: %s", t.tokenType, t.val)
	}

	for {
		next, ok := p.nextToken()
		if !ok {
			return fmt.Errorf("parsing names got eof")
		}

		// TODO is this needed for shadowing?
		t = next

		if t.tokenType == RPAREN {
			p.state = EXPECT_KEYWORD
			return nil
		}

		if t.tokenType != QDESCR {
			return fmt.Errorf("expected qdescr or rparen, got %s: %s", t.tokenType, t.val)
		}

		p.opts = append(p.opts, schema.WithName(stripQuotes(t.val)))
	}
}

func (p *objectClassParser) parseDesc() error {
	t, ok := p.nextToken()
	if !ok {
		return fmt.Errorf("parsing desc got eof")
	}

	// qdescr is a subset of qdstring, the tokeniser will match qdescr first - so allow both
	if t.tokenType != QDESCR && t.tokenType != QDSTRING {
		return fmt.Errorf("expected qdescr or qdstring, got %s: %s", t.tokenType, t.val)
	}

	p.opts = append(p.opts, schema.WithDesc(stripQuotes(t.val)))

	p.state = EXPECT_KEYWORD
	return nil
}

func (p *objectClassParser) parseOids() ([]dit.OID, error) {
	t, ok := p.nextToken()
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
		next, ok := p.nextToken()
		if !ok {
			return oids, fmt.Errorf("parsing names got eof")
		}

		// skip $ separators
		if next.tokenType == DOLLAR {
			if next, ok = p.nextToken(); !ok {
				return oids, fmt.Errorf("parsing names got eof")
			}
		}

		// TODO is this needed for shadowing?
		t = next

		if t.tokenType == RPAREN {
			p.state = EXPECT_KEYWORD
			return oids, nil
		}

		if t.tokenType != DESCR && t.tokenType != KEYWORD {
			return oids, fmt.Errorf("expected qdescr or rparen, got %s: %s", t.tokenType, t.val)
		}

		oids = append(oids, dit.OID(t.val))
	}
}

func (p *objectClassParser) parseNextToken() error {
	switch p.state {
	case EXPECT_NUMERICOID:
		return p.parseNumericoid()
	case EXPECT_KEYWORD:
		return p.parseKeyword()
	case EXPECT_NAME:
		return p.parseName()
	case EXPECT_DESC:
		return p.parseDesc()
	case EXPECT_SUP:
		oids, err := p.parseOids()
		if err != nil {
			return err
		}
		p.opts = append(p.opts, schema.WithSupOid(oids...))
		return nil
	case EXPECT_MUST:
		oids, err := p.parseOids()
		if err != nil {
			return err
		}
		p.opts = append(p.opts, schema.WithMustAttr(oids...))
		return nil
	case EXPECT_MAY:
		oids, err := p.parseOids()
		if err != nil {
			return err
		}
		p.opts = append(p.opts, schema.WithMayAttr(oids...))
		return nil
	default:
		return fmt.Errorf("unknown or unimplemented p state %s", p.state)
	}
}

func (p *objectClassParser) parse() (schema.ObjectClass, error) {
	for {
		err := p.parseNextToken()
		if err == io.EOF {
			break
		}
		if err != nil {
			return schema.ObjectClass{}, err
		}
	}

	return schema.NewObjectClass(p.opts...), nil
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

func ParseObjectClasses(r io.Reader) ([]schema.ObjectClass, error) {
	tokens, err := tokenise(r)
	if err != nil {
		return nil, err
	}

	if !checkTokensBalanced(tokens) {
		return nil, fmt.Errorf("Tokens unbalanced")
	}

	ocs := []schema.ObjectClass{}

	idx := 0
	for idx < len(tokens) {
		end, err := findMatchingParen(tokens, idx)
		if err != nil {
			return nil, err
		}

		ocTokens := tokens[idx+1 : end-1]
		log.Print("---- oc tokens ----")
		for _, t := range ocTokens {
			log.Printf("%s: %s", t.tokenType, t.val)
		}
		log.Print("---- end oc tokens ----")
		oc, err := newObjectClassParser(ocTokens).parse()
		if err != nil {
			return nil, err
		}
		ocs = append(ocs, oc)
		idx = end
	}

	return ocs, nil
}
