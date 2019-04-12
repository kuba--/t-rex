// Package trex is a Tiny Regular EXpressions implementation,
// heavilly inspired by "A Regular Expression Matcher" -
// http://www.cs.princeton.edu/courses/archive/spr09/cos333/beautiful.html
// and https://github.com/monolifed/tiny-regex-mod
// Supports:
//   '^'        Start anchor, matches start of string
//   '$'        End anchor, matches end of string
//   '*'        Asterisk, match zero or more (greedy, *? lazy)
//   '+'        Plus, match one or more (greedy, +? lazy)
//   '{m,n}'    Quantifier, match min. 'm' and max. 'n' (greedy, {m,n}? lazy)
//   '{m}'                  exactly 'm'
//   '{m,}'                 match min 'm' and max. MAX_QUANT
//   '?'        Question, match zero or one (greedy, ?? lazy)
//   '.'        Dot, matches any character except newline (\r, \n)
//   '[abc]'    Character class, match if one of {'a', 'b', 'c'}
//   '[^abc]'   Inverted class, match if NOT one of {'a', 'b', 'c'}
//   '[a-zA-Z]' Character ranges, the character set of the ranges { a-z | A-Z }
//   '\s'       Whitespace, \t \f \r \n \v and spaces
//   '\S'       Non-whitespace
//   '\w'       Alphanumeric, [a-zA-Z0-9_]
//   '\W'       Non-alphanumeric
//   '\d'       Digits, [0-9]
//   '\D'       Non-digits
//   '\X'       Character itself; X in [^sSwWdD] (e.g. '\\' is '\')
package trex

import (
	"bytes"
	"fmt"
	"strings"
)

type Type int

const (
	None Type = iota
	Begin
	End
	Quant
	LQuant
	QMark
	LQMark
	Star
	LStar
	Plus
	LPlus
	Dot
	Char
	Class
	NClass
	Digit
	NDigit
	Alpha
	NAlpha
	Space
	NSpace
)

func (t Type) String() string {
	return []string{
		"None",
		"Begin",
		"End",
		"Quant",
		"LQuant",
		"QMark",
		"LQMark",
		"Star",
		"LStar",
		"Plus",
		"LPlus",
		"Dot",
		"Char",
		"Class",
		"NClass",
		"Digit",
		"NDigit",
		"Alpha",
		"NAlpha",
		"Space",
		"NSpace",
	}[t]
}

const (
	MaxNodes  = 64
	MaxBufLen = 128
	MaxQuant  = 1024
	MaxPlus   = 40000
)

type node struct {
	typ Type

	ch  byte
	ccl []byte
	mn  [2]int
}

type Regexp struct {
	nodes  []node
	buffer []byte
}

func Compile(expr string) (*Regexp, error) {
	return compile([]byte(expr))
}

func compile(expr []byte) (*Regexp, error) {
	n := len(expr)
	if n == 0 {
		return nil, fmt.Errorf("An empty expression string")
	}

	// is the last node quantifiable
	var quable bool

	re := &Regexp{
		nodes:  make([]node, MaxNodes),
		buffer: make([]byte, MaxBufLen),
	}
	j, idx := 0, 0
	for i := 0; i < n && (j+1 < MaxNodes); i, j = i+1, j+1 {
		switch expr[i] {
		// Meta-characters
		case '^':
			quable = false
			re.nodes[j].typ = Begin

		case '$':
			quable = false
			re.nodes[j].typ = End

		case '.':
			quable = true
			re.nodes[j].typ = Dot

		case '*':
			if !quable {
				return nil, fmt.Errorf("Non-quantifiable before *")
			}
			quable = false
			if ii := i + 1; ii < n && expr[ii] == '?' {
				i = ii
				re.nodes[j].typ = LStar
			} else {
				re.nodes[j].typ = Star
			}

		case '+':
			if !quable {
				return nil, fmt.Errorf("Non-quantifiable before +")
			}
			quable = false
			if ii := i + 1; ii < n && expr[ii] == '?' {
				i = ii
				re.nodes[j].typ = LPlus
			} else {
				re.nodes[j].typ = Plus
			}

		case '?':
			if !quable {
				return nil, fmt.Errorf("Non-quantifiable before ?")
			}
			quable = false
			if ii := i + 1; ii < n && expr[ii] == '?' {
				i = ii
				re.nodes[j].typ = LQMark
			} else {
				re.nodes[j].typ = QMark
			}

		case '\\':
			quable = true
			i++
			if i >= n {
				return nil, fmt.Errorf("Dangling \\")
			}
			switch expr[i] {
			case 'd':
				re.nodes[j].typ = Digit
			case 'D':
				re.nodes[j].typ = NDigit
			case 'w':
				re.nodes[j].typ = Alpha
			case 'W':
				re.nodes[j].typ = NAlpha
			case 's':
				re.nodes[j].typ = Space
			case 'S':
				re.nodes[j].typ = NSpace
			default:
				re.nodes[j].typ = Char
				re.nodes[j].ch = expr[i]
			}

		// Character class
		case '[':
			quable = true
			// Look-ahead to determine if negated
			if ii := i + 1; ii < n && expr[ii] == '^' {
				i = ii
				re.nodes[j].typ = NClass
			} else {
				re.nodes[j].typ = Class
			}
			re.nodes[j].ccl = re.buffer[idx:]

			// Copy characters inside [..] to buffer
			for i++; i < n && expr[i] != ']'; i++ {
				if expr[i] == '\\' {
					ii := i + 1
					if ii >= n {
						return nil, fmt.Errorf("Dangling \\ in class")
					}
					// needs escaping ?
					if isMetaOrEsc(expr[ii]) {
						if idx > MaxBufLen-3 {
							return nil, fmt.Errorf("Buffer overflow at <esc>char in class")
						}

						re.buffer[idx] = expr[i]
						idx++
						i = ii

						re.buffer[idx] = expr[i]
						idx++
						if expr[i+1] != '\\' {
							continue
						}
					} else { // skip esc
						if idx > MaxBufLen-2 {
							return nil, fmt.Errorf("Buffer overflow at [esc]char in class")
						}
						i++
						re.buffer[idx] = expr[i]
						idx++
					}
				} else {
					if idx > MaxBufLen-2 {
						return nil, fmt.Errorf("Buffer overflow at [esc]char in class")
					}
					re.buffer[idx] = expr[i]
					idx++
				}

				// check range
				if expr[i+1] != '-' || i+2 >= n || expr[i+2] == ']' {
					continue
				}

				rmax := '\\' == expr[i+2]
				if rmax && (i+3 >= n || isMeta(expr[i+3])) {
					continue
				}

				var c byte
				if rmax {
					c = expr[i+3]
				} else {
					c = expr[i+2]
				}
				if c < expr[i] {
					return nil, fmt.Errorf("Incorrect range in class")
				}
				if idx > MaxBufLen-2 {
					return nil, fmt.Errorf("Buffer overflow at range - in class")
				}

				i++
				re.buffer[idx] = expr[i] // '-'
				idx++
			}

			if expr[i] != ']' {
				return nil, fmt.Errorf("Non terminated class")
			}
			// // Nul-terminated string
			re.buffer[idx] = 0
			idx++

		case '{':
			if !quable {
				return nil, fmt.Errorf("Non-quantifiable before {m,n}")
			}
			quable = false

			i++
			var val int
			for {
				if i >= n || expr[i] < '0' || expr[i] > '9' {
					return nil, fmt.Errorf("Non-digit in quantifier min value")
				}
				val = 10*val + int(expr[i]-'0')
				i++

				if expr[i] == ',' || expr[i] == '}' {
					break
				}
			}

			if val > MaxQuant {
				return nil, fmt.Errorf("Quantifier min value too big")
			}
			re.nodes[j].mn[0] = val

			if expr[i] == ',' {
				i++
				if i >= n {
					return nil, fmt.Errorf("Unexpected end of string in quantifier")
				}
				if expr[i] == '}' {
					val = MaxQuant
				} else {
					val = 0
					for expr[i] != '}' {
						if i >= n || expr[i] < '0' || expr[i] > '9' {
							return nil, fmt.Errorf("Non-digit in quantifier max value")
						}
						val = 10*val + int(expr[i]-'0')
						i++
					}
					if val > MaxQuant || val < re.nodes[j].mn[0] {
						return nil, fmt.Errorf("Quantifier max value too big or less than min value")
					}
				}
			}
			if ii := i + 1; ii < n && expr[ii] == '?' {
				i++
				re.nodes[j].typ = LQuant
			} else {
				re.nodes[j].typ = Quant
			}
			re.nodes[j].mn[1] = val

		default:
			quable = true
			re.nodes[j].typ = Char
			re.nodes[j].ch = expr[i]
		}

	}
	// None used to indicate end-of-pattern
	re.nodes[j].typ = None
	return re, nil
}

func isMeta(c byte) bool {
	return c == 's' || c == 'S' || c == 'w' || c == 'W' || c == 'd' || c == 'D'
}

func isMetaOrEsc(c byte) bool {
	return c == '\\' || isMeta(c)
}

func (re *Regexp) Match(b []byte) bool {
	n := len(b)
	if n == 0 {
		return false
	}

	nodes := re.nodes
	if nodes[0].typ == Begin {
		return match(nodes[1:], b)
	}

	for len(b) > 0 {
		if match(nodes, b) {
			return true
		}
		b = b[1:]
	}
	return false
}

func match(nodes []node, txt []byte) bool {
	for {
		if nodes[0].typ == None {
			return true
		}

		if nodes[0].typ == End && nodes[1].typ == None {
			return len(txt) == 0
		}

		switch nodes[1].typ {
		case QMark:
			return matchQuant(nodes, txt, 0, 1)
		case LQMark:
			return matchLQuant(nodes, txt, 0, 1)
		case Quant:
			return matchQuant(nodes, txt, nodes[1].mn[0], nodes[1].mn[1])
		case LQuant:
			return matchLQuant(nodes, txt, nodes[1].mn[0], nodes[1].mn[1])
		case Star:
			return matchQuant(nodes, txt, 0, MaxPlus)
		case LStar:
			return matchLQuant(nodes, txt, 0, MaxPlus)
		case Plus:
			return matchQuant(nodes, txt, 1, MaxPlus)
		case LPlus:
			return matchLQuant(nodes, txt, 1, MaxPlus)
		}

		if len(txt) == 0 || !matchOne(nodes[0], txt[0]) {
			break
		}
		nodes = nodes[1:]
		txt = txt[1:]
	}
	return false
}

func matchQuant(nodes []node, txt []byte, min, max int) bool {
	i := 0
	for max != 0 && i < len(txt) && matchOne(nodes[0], txt[i]) {
		i++
		max--
	}

	nn := nodes[2:]
	for i >= min {
		if match(nn, txt[i:]) {
			return true
		}
		i--
	}

	return false
}

func matchLQuant(nodes []node, txt []byte, min, max int) bool {
	max = max - min + 1
	i := 0
	for min != 0 && i < len(txt) && matchOne(nodes[0], txt[i]) {
		i++
		min--
	}
	if min != 0 {
		return false
	}

	nn := nodes[2:]
	txt = txt[i:]
	for {
		if match(nn, txt) {
			return true
		}
		max--

		if max == 0 || len(txt) == 0 || !matchOne(nodes[0], txt[0]) {
			break
		}
		txt = txt[1:]
	}

	return false
}

func matchOne(n node, b byte) bool {
	switch n.typ {
	case Char:
		return (n.ch == b)
	case Dot:
		return matchDot(b)
	case Class:
		return matchCharClass(b, n.ccl)
	case NClass:
		return !matchCharClass(b, n.ccl)
	case Digit:
		return matchDigit(b)
	case NDigit:
		return !matchDigit(b)
	case Alpha:
		return matchAlphaNum(b)
	case NAlpha:
		return !matchAlphaNum(b)
	case Space:
		return matchSpace(b)
	case NSpace:
		return !matchSpace(b)
	}
	return false
}

func matchCharClass(b byte, txt []byte) bool {
	var rmax byte
	str := txt
	if i := bytes.IndexByte(str, 0); i > 0 {
		str = str[0 : i+1]
	}

	for i := 0; str[0] != 0; {
		if str[0] == '\\' {
			if matchMetaChar(b, str[1]) {
				return true
			}
			i += 2
			str = str[2:]

			if isMeta(str[0]) {
				continue
			}
		} else {
			if str[0] == b {
				return true
			}
			i++
			str = str[1:]
		}

		if str[0] != '-' || str[1] == 0 {
			continue
		}

		if str[1] == '\\' && isMeta(str[2]) {
			continue
		}

		if str[1] == '\\' {
			rmax = str[2]
		} else {
			rmax = str[1]
		}

		if b >= txt[i-1] && b <= rmax {
			return true
		}
		i++
		str = str[1:]
	}
	return false
}

func matchMetaChar(b, mb byte) bool {
	switch mb {
	case 'd':
		return matchDigit(b)
	case 'D':
		return !matchDigit(b)
	case 'w':
		return matchAlphaNum(b)
	case 'W':
		return !matchAlphaNum(b)
	case 's':
		return matchSpace(b)
	case 'S':
		return !matchSpace(b)
	}

	return b == mb
}

func matchDot(b byte) bool {
	return b != '\n' && b != '\r'
}

func matchDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func matchAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func matchAlphaNum(b byte) bool {
	return b == '_' || matchAlpha(b) || matchDigit(b)
}

func matchSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f' || b == '\v'
}

func (re *Regexp) String() string {
	sb := &strings.Builder{}
	for _, n := range re.nodes {
		if n.typ == None {
			break
		}
		sb.WriteString(fmt.Sprintf("type: %s", n.typ.String()))
		switch n.typ {
		case Class, NClass:
			if i := bytes.IndexByte(n.ccl, 0); i > 0 {
				sb.WriteString(fmt.Sprintf(" \"%s\"", n.ccl[:i]))
			} else {
				sb.WriteString(fmt.Sprintf(" \"%s\"", n.ccl))
			}

		case Quant, LQuant:
			sb.WriteString(fmt.Sprintf(" {%d, %d}", n.mn[0], n.mn[1]))

		case Char:
			sb.WriteString(fmt.Sprintf(" '%c'", n.ch))
		}
		sb.WriteByte('\n')
	}

	return sb.String()
}
