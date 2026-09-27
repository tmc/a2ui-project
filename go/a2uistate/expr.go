package a2uistate

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Limits on formatString templates, as in the web renderers.
const (
	maxTemplateLen   = 10000 // UTF-16 code units
	maxTemplateParts = 1000
	maxParseDepth    = 100
)

var errParseDepth = errors.New("expressions nested too deeply")

// parseTemplate parses a formatString template into its parts: literal
// strings and the dynamic values of its ${...} expressions, in the JSON
// form that Evaluator.eval takes. It follows the expression parser of
// web_core, but also accepts function names that start with @.
func parseTemplate(s string, depth int) ([]any, error) {
	if depth > maxParseDepth {
		return nil, errParseDepth
	}
	if n := utf16Len(s); n > maxTemplateLen {
		return nil, fmt.Errorf("template length %d exceeds %d", n, maxTemplateLen)
	}
	if s == "" {
		return nil, nil
	}
	if !strings.Contains(s, "${") {
		return []any{s}, nil
	}
	var parts []any
	sc := &scanner{s: s}
	for !sc.atEnd() {
		if len(parts) >= maxTemplateParts {
			return nil, fmt.Errorf("template has more than %d parts", maxTemplateParts)
		}
		switch {
		case sc.hasPrefix("${"):
			sc.advance(2)
			content, err := sc.interpolation()
			if err != nil {
				return nil, err
			}
			v, err := parseExpression(content, depth+1)
			if err != nil {
				return nil, err
			}
			parts = append(parts, v)
		case sc.hasPrefix(`\${`):
			sc.advance(3)
			parts = append(parts, "${")
		default:
			start := sc.pos
			for !sc.atEnd() && !sc.hasPrefix("${") && !sc.hasPrefix(`\${`) {
				sc.advance(1)
			}
			parts = append(parts, s[start:sc.pos])
		}
	}
	var out []any
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out, nil
}

// parseExpression parses the content of ${...}, which is one
// expression.
func parseExpression(expr string, depth int) (any, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", nil
	}
	sc := &scanner{s: expr}
	v, err := sc.expression(depth)
	if err != nil {
		return nil, err
	}
	if !sc.atEnd() {
		return nil, fmt.Errorf("unexpected characters at end of expression: %q", expr[sc.pos:])
	}
	return v, nil
}

// A scanner scans a template or expression byte by byte. All the
// syntax is ASCII, so this matches web_core, which scans UTF-16 code
// units.
type scanner struct {
	s   string
	pos int
}

func (sc *scanner) atEnd() bool { return sc.pos >= len(sc.s) }

// peek returns the byte at offset from the current position, or 0.
func (sc *scanner) peek(offset int) byte {
	if i := sc.pos + offset; i < len(sc.s) {
		return sc.s[i]
	}
	return 0
}

// advance moves past n bytes and returns the first, or 0 at the end.
func (sc *scanner) advance(n int) byte {
	c := sc.peek(0)
	sc.pos += n
	return c
}

func (sc *scanner) hasPrefix(p string) bool {
	return sc.pos < len(sc.s) && strings.HasPrefix(sc.s[sc.pos:], p)
}

func (sc *scanner) match(c byte) bool {
	if !sc.atEnd() && sc.peek(0) == c {
		sc.pos++
		return true
	}
	return false
}

func (sc *scanner) skipSpace() {
	for !sc.atEnd() && strings.IndexByte(" \t\n\r\v\f", sc.peek(0)) >= 0 {
		sc.pos++
	}
}

// keyword reports whether the keyword kw is next, and moves past it.
func (sc *scanner) keyword(kw string) bool {
	if !sc.hasPrefix(kw) {
		return false
	}
	if c := sc.peek(len(kw)); isAlnum(c) || c == '_' {
		return false
	}
	sc.pos += len(kw)
	return true
}

// interpolation returns the content of an interpolation, whose "${"
// has been scanned, and moves past its closing brace.
func (sc *scanner) interpolation() (string, error) {
	start := sc.pos
	depth := 1
	for !sc.atEnd() && depth > 0 {
		switch c := sc.advance(1); c {
		case '{':
			depth++
		case '}':
			depth--
		case '\'', '"':
			for !sc.atEnd() {
				d := sc.advance(1)
				if d == '\\' {
					sc.advance(1)
				} else if d == c {
					break
				}
			}
		}
	}
	if depth > 0 {
		return "", errors.New("unclosed interpolation: missing '}'")
	}
	return sc.s[start : sc.pos-1], nil
}

// expression parses one expression: a nested interpolation, a literal,
// a function call or a data binding path.
func (sc *scanner) expression(depth int) (any, error) {
	if depth > maxParseDepth {
		return nil, errParseDepth
	}
	sc.skipSpace()
	if sc.atEnd() {
		return "", nil
	}
	c := sc.peek(0)
	switch {
	case sc.hasPrefix("${"):
		sc.advance(2)
		content, err := sc.interpolation()
		if err != nil {
			return nil, err
		}
		return parseExpression(content, depth+1)
	case c == '\'' || c == '"':
		return sc.stringLiteral(), nil
	case isDigit(c) || (c == '-' || c == '+') && isDigit(sc.peek(1)):
		return sc.number()
	case sc.keyword("true"):
		return true, nil
	case sc.keyword("false"):
		return false, nil
	case sc.keyword("null"):
		return "", nil
	}
	start := sc.pos
	if c == '@' {
		sc.pos++
	}
	for !sc.atEnd() {
		c := sc.peek(0)
		if !isAlnum(c) && c != '/' && c != '.' && c != '_' && c != '-' {
			break
		}
		sc.pos++
	}
	token := sc.s[start:sc.pos]
	sc.skipSpace()
	if sc.peek(0) == '(' {
		return sc.call(token, depth)
	}
	if token == "" || token[0] == '@' {
		sc.pos = start
		return "", nil
	}
	return map[string]any{"path": token}, nil
}

// call parses the arguments of a call to the function name.
func (sc *scanner) call(name string, depth int) (any, error) {
	sc.match('(')
	sc.skipSpace()
	args := make(map[string]any)
	for !sc.atEnd() && sc.peek(0) != ')' {
		start := sc.pos
		for !sc.atEnd() && (isAlnum(sc.peek(0)) || sc.peek(0) == '_') {
			sc.pos++
		}
		arg := sc.s[start:sc.pos]
		sc.skipSpace()
		if !sc.match(':') {
			return nil, fmt.Errorf("expected ':' after argument name %q in function %q", arg, name)
		}
		sc.skipSpace()
		v, err := sc.expression(depth + 1)
		if err != nil {
			return nil, err
		}
		args[arg] = v
		sc.skipSpace()
		if sc.match(',') {
			sc.skipSpace()
		}
	}
	if !sc.match(')') {
		return nil, fmt.Errorf("expected ')' after function arguments for %q", name)
	}
	return map[string]any{"call": name, "args": args}, nil
}

// stringLiteral parses a quoted string, with the escapes \n, \t, \r
// and \ before any other character.
func (sc *scanner) stringLiteral() string {
	quote := sc.advance(1)
	var b strings.Builder
	for !sc.atEnd() {
		c := sc.advance(1)
		if c == '\\' {
			switch d := sc.advance(1); d {
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case 0:
			default:
				b.WriteByte(d)
			}
			continue
		}
		if c == quote {
			break
		}
		b.WriteByte(c)
	}
	return b.String()
}

var numberLiteral = regexp.MustCompile(`^[+-]?\d+\.?\d*(?:[eE][+-]?\d+)?$`)

// number parses a number literal.
func (sc *scanner) number() (any, error) {
	start := sc.pos
	if c := sc.peek(0); c == '-' || c == '+' {
		sc.pos++
	}
	for !sc.atEnd() && (isDigit(sc.peek(0)) || sc.peek(0) == '.') {
		sc.pos++
	}
	if c := sc.peek(0); !sc.atEnd() && (c == 'e' || c == 'E') {
		sc.pos++
		if c := sc.peek(0); !sc.atEnd() && (c == '+' || c == '-') {
			sc.pos++
		}
		for !sc.atEnd() && isDigit(sc.peek(0)) {
			sc.pos++
		}
	}
	text := sc.s[start:sc.pos]
	if !numberLiteral.MatchString(text) {
		return nil, fmt.Errorf("invalid number literal %q", text)
	}
	return strconv.ParseFloat(text, 64)
}

func isDigit(c byte) bool { return '0' <= c && c <= '9' }

func isAlnum(c byte) bool {
	return isDigit(c) || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}
