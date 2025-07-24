package ldif

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
)

var logger = log.New(os.Stderr, "parser: ", log.Lshortfile)

type TokenType int

const (
	NUMERICOID TokenType = iota
	DESCR
	NOIDLEN
	LPAREN
	RPAREN
	LCURLY
	RCURLY
	QDESCR
	QDSTRING
	KEYWORD
	DOLLAR
)

func (t TokenType) String() string {
	switch t {
	case NUMERICOID:
		return "NUMERICOID"
	case DESCR:
		return "DESCR"
	case NOIDLEN:
		return "NOIDLEN"
	case LPAREN:
		return "LPAREN"
	case RPAREN:
		return "RPAREN"
	case LCURLY:
		return "LCURLY"
	case RCURLY:
		return "RCURLY"
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
	keyword_re    = regexp.MustCompile(`^[A-Z][A-Z-]*$`)
	noidlen_re    = regexp.MustCompile(`^[0-9]+(\.[0-9]+)+({[1-9][0-9]*})?$`)
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
	case val == "{":
		return LCURLY, nil
	case val == "}":
		return RCURLY, nil
	case val == "$":
		return DOLLAR, nil
	case numericoid_re.Match([]byte(val)):
		return NUMERICOID, nil
	case noidlen_re.Match([]byte(val)):
		return NOIDLEN, nil
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

type Tokeniser struct {
	tokens []Token
}

func NewTokeniser(r io.Reader) (*Tokeniser, error) {
	tokens, err := tokenise(r)
	if err != nil {
		return nil, err
	}

	if !checkTokensBalanced(tokens) {
		return nil, fmt.Errorf("tokens unbalanced")
	}

	return &Tokeniser{tokens}, nil
}

func (t *Tokeniser) HasNext() bool {
	return len(t.tokens) > 0
}

func (t *Tokeniser) Next() (Token, bool) {
	if len(t.tokens) == 0 {
		return Token{}, false
	}

	token := t.tokens[0]
	t.tokens = t.tokens[1:]
	return token, true
}

func (t *Tokeniser) Peek() (Token, bool) {
	if len(t.tokens) == 0 {
		return Token{}, false
	}
	return t.tokens[0], true
}

func (t *Tokeniser) NextNumericoid() (Token, error) {
	n, ok := t.Next()
	if !ok {
		return Token{}, fmt.Errorf("expected numericoid, got nothing")
	}

	if n.tokenType != NUMERICOID {
		return Token{}, fmt.Errorf("expected numericoid, got %s (%s)", n.tokenType, n.val)
	}

	return n, nil
}

func (t *Tokeniser) NextOid() (Token, error) {
	n, ok := t.Next()
	if !ok {
		return Token{}, fmt.Errorf("expected oid, got nothing")
	}

	if n.tokenType != NUMERICOID && n.tokenType != KEYWORD && n.tokenType != DESCR {
		return Token{}, fmt.Errorf("expected oid, got %s (%s)", n.tokenType, n.val)
	}

	return n, nil
}

func (t *Tokeniser) NextOids() ([]Token, error) {
	peek, ok := t.Peek()
	if !ok {
		return nil, fmt.Errorf("expected oids, got nothing")
	}

	if peek.tokenType == NUMERICOID || peek.tokenType == KEYWORD || peek.tokenType == DESCR {
		n, _ := t.Next()
		return []Token{n}, nil
	}

	subt, err := t.ParenSubTokeniser()
	if err != nil {
		return nil, err
	}

	tokens := []Token{}
	dollar := false
	for on, ok := subt.Next(); ok; on, ok = subt.Next() {
		if dollar {
			if on.tokenType == DOLLAR {
				dollar = !dollar
				continue
			}
			return nil, fmt.Errorf("expected dollar, got %q (%q)", on.tokenType, on.val)
		}

		if on.tokenType != NUMERICOID && on.tokenType != KEYWORD && on.tokenType != DESCR {
			return nil, fmt.Errorf("expected oid got %s", on.tokenType)
		}

		tokens = append(tokens, on)
		dollar = !dollar
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("oids is empty")
	}

	return tokens, nil
}

func (t *Tokeniser) NextNoidlen() (Token, error) {
	n, ok := t.Next()
	if !ok {
		return Token{}, fmt.Errorf("expected noidlen, got nothing")
	}

	if n.tokenType != NUMERICOID && n.tokenType != NOIDLEN {
		return Token{}, fmt.Errorf("expected noidlen, got %s (%s)", n.tokenType, n.val)
	}

	return n, nil
}

func (t *Tokeniser) NextQdescrs() ([]Token, error) {
	p, ok := t.Peek()
	if !ok {
		return nil, fmt.Errorf("expected qdescr(s), got nothing")
	}

	if p.tokenType == QDESCR {
		n, _ := t.Next()
		return []Token{n}, nil
	}

	qt, err := t.ParenSubTokeniser()
	if err != nil {
		return nil, err
	}

	tokens := []Token{}
	for qn, ok := qt.Next(); ok; qn, ok = qt.Next() {
		if qn.tokenType != QDESCR {
			return nil, fmt.Errorf("expected qdescr got %s", qn.tokenType)
		}

		tokens = append(tokens, qn)
	}

	if len(tokens) == 0 {
		return nil, fmt.Errorf("qdescrs is empty")
	}

	return tokens, nil
}

func (t *Tokeniser) NextQdstring() (Token, error) {
	n, ok := t.Next()
	if !ok {
		return Token{}, fmt.Errorf("expected qdstring, got nothing")
	}

	if n.tokenType != QDESCR && n.tokenType != QDSTRING {
		return Token{}, fmt.Errorf("expected qdstring, got %s", n.tokenType)
	}

	return n, nil
}

func (t *Tokeniser) NextDescr() (Token, error) {
	n, ok := t.Next()
	if !ok {
		return Token{}, fmt.Errorf("expected descr, got nothing")
	}

	if n.tokenType != DESCR {
		return Token{}, fmt.Errorf("expected descr, got %s", n.tokenType)
	}

	return n, nil
}

// returns a subtokeniser with just the tokens inside the parentheses of the current token
// removes the tokens from the main tokeniser
// errors if the main tokeniser is not on a LPAREN
func (t *Tokeniser) ParenSubTokeniser() (*Tokeniser, error) {
	if t.tokens[0].tokenType != LPAREN {
		return nil, fmt.Errorf("Expected LPAREN, got %s: %s",
			t.tokens[0].tokenType,
			t.tokens[0].val,
		)
	}

	bal := 1
	rparenIdx := 1
	for ; rparenIdx < len(t.tokens) && bal != 0; rparenIdx += 1 {
		if t.tokens[rparenIdx].tokenType == LPAREN {
			bal += 1
		}

		if t.tokens[rparenIdx].tokenType == RPAREN {
			bal -= 1
		}
	}

	if bal != 0 {
		return nil, fmt.Errorf("Could not find matching rparen")
	}

	subtokeniser := &Tokeniser{tokens: t.tokens[1 : rparenIdx-1]}
	t.tokens = t.tokens[rparenIdx:]

	return subtokeniser, nil
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
		// if at any point there are more right parens then left then something is wrong
		if r > l {
			return false
		}
		if t.tokenType == LPAREN {
			l += 1
		}
		if t.tokenType == RPAREN {
			r += 1
		}
	}

	return l == r
}

func stripQuotes(s string) string {
	if s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}
	return s
}
