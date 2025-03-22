package ldif

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
)

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
	keyword_re    = regexp.MustCompile(`^[A-Z]+$`)
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
