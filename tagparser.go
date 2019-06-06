package tagexpr

import (
	"fmt"
	"strings"
	"unicode"
)

func (f *fieldVM) parseExprs(tag string) error {
	raw := tag
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return nil
	}
	if tag[0] != '{' {
		expr, err := parseExpr(tag)
		if err != nil {
			return err
		}
		exprSelector := f.structField.Name
		f.exprs[exprSelector] = expr
		f.origin.exprs[exprSelector] = expr
		f.origin.exprSelectorList = append(f.origin.exprSelectorList, exprSelector)
		return nil
	}
	var subtag *string
	var idx int
	var exprSelector, exprStr string
	for {
		subtag = readPairedSymbol(&tag, '{', '}')
		if subtag != nil {
			idx = strings.Index(*subtag, ":")
			if idx > 0 {
				exprSelector = strings.TrimSpace((*subtag)[:idx])
				switch exprSelector {
				case "":
					continue
				case ExprNameSeparator:
					exprSelector = f.structField.Name
				default:
					exprSelector = f.structField.Name + ExprNameSeparator + exprSelector
				}
				if _, had := f.origin.exprs[exprSelector]; had {
					return fmt.Errorf("duplicate expression name: %s", exprSelector)
				}
				exprStr = strings.TrimSpace((*subtag)[idx+1:])
				if exprStr != "" {
					if expr, err := parseExpr(exprStr); err == nil {
						f.exprs[exprSelector] = expr
						f.origin.exprs[exprSelector] = expr
						f.origin.exprSelectorList = append(f.origin.exprSelectorList, exprSelector)
					} else {
						return err
					}
					trimLeftSpace(&tag)
					if tag == "" {
						return nil
					}
					continue
				}
			}
		}
		return fmt.Errorf("syntax incorrect: %q", raw)
	}
}

func trimLeftSpace(p *string) *string {
	*p = strings.TrimLeftFunc(*p, unicode.IsSpace)
	return p
}

func readPairedSymbol(p *string, left, right rune) *string {
	s := *p
	if len(s) == 0 || rune(s[0]) != left {
		return nil
	}
	s = s[1:]
	var last1 = left
	var last2 rune
	var leftLevel, rightLevel int
	var escapeIndexes = make(map[int]bool)
	var realEqual, escapeEqual bool
	for i, r := range s {
		if realEqual, escapeEqual = equalRune(right, r, last1, last2); realEqual {
			if leftLevel == rightLevel {
				*p = s[i+1:]
				var sub = make([]rune, 0, i)
				for k, v := range s[:i] {
					if !escapeIndexes[k] {
						sub = append(sub, v)
					}
				}
				s = string(sub)
				return &s
			}
			rightLevel++
		} else if escapeEqual {
			escapeIndexes[i-1] = true
		} else if realEqual, escapeEqual = equalRune(left, r, last1, last2); realEqual {
			leftLevel++
		} else if escapeEqual {
			escapeIndexes[i-1] = true
		}
		last2 = last1
		last1 = r
	}
	return nil
}

func equalRune(a, b, last1, last2 rune) (real, escape bool) {
	if a == b {
		real = last1 != '\\' || last2 == '\\'
		escape = last1 == '\\' && last2 != '\\'
	}
	return
}
