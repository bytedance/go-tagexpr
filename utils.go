package tagexpr

import (
	"strings"
	"unicode"
)

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
	for i, r := range s {
		if r == right && (last1 != '\\' || last2 == '\\') {
			if leftLevel == rightLevel {
				*p = s[i+1:]
				sub := s[:i]
				return &sub
			}
			rightLevel++
		} else if r == left && (last1 != '\\' || last2 == '\\') {
			leftLevel++
		}
		last2 = last1
		last1 = r
	}
	return nil
}
