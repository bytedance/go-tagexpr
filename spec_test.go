package tagexpr

import (
	"reflect"
	"testing"
)

func TestReadPairedSymbol(t *testing.T) {
	var cases = []struct {
		expr         string
		val          string
		lastExprNode string
		left, right  rune
	}{
		{expr: "'true '+'a'", val: "true ", lastExprNode: "+'a'", left: '\'', right: '\''},
		{expr: "((0+1)/(2-1)*9)%2", val: "(0+1)/(2-1)*9", lastExprNode: "%2", left: '(', right: ')'},
	}
	for _, c := range cases {
		t.Log(c.expr)
		expr := c.expr
		got := readPairedSymbol(&expr, c.left, c.right)
		if *got != c.val || expr != c.lastExprNode {
			t.Fatalf("expr: %q, got: %q, %q, want: %q, %q", c.expr, *got, expr, c.val, c.lastExprNode)
		}
	}
}

func TestReadBoolExprNode(t *testing.T) {
	var cases = []struct {
		expr         string
		val          bool
		lastExprNode string
	}{
		{expr: "false", val: false, lastExprNode: ""},
		{expr: "true", val: true, lastExprNode: ""},
		{expr: "true ", val: true, lastExprNode: " "},
		{expr: "!true&", val: false, lastExprNode: "&"},
		{expr: "!false|", val: true, lastExprNode: "|"},
		{expr: "!!!!false =", val: !!!!false, lastExprNode: " ="},
	}
	for _, c := range cases {
		t.Log(c.expr)
		expr := c.expr
		e := readBoolExprNode(&expr)
		got := e.Run(nil).(bool)
		if got != c.val || expr != c.lastExprNode {
			t.Fatalf("expr: %s, got: %v, %s, want: %v, %s", c.expr, got, expr, c.val, c.lastExprNode)
		}
	}
}

func TestReadDigitalExprNode(t *testing.T) {
	var cases = []struct {
		expr         string
		val          float64
		lastExprNode string
	}{
		{expr: "0.1 +1", val: 0.1, lastExprNode: " +1"},
		{expr: "-1\\1", val: -1, lastExprNode: "\\1"},
		{expr: "1a", val: 0, lastExprNode: ""},
		{expr: "1", val: 1, lastExprNode: ""},
		{expr: "1.1", val: 1.1, lastExprNode: ""},
		{expr: "1.1/", val: 1.1, lastExprNode: "/"},
	}
	for _, c := range cases {
		expr := c.expr
		e := readDigitalExprNode(&expr)
		if c.expr == "1a" {
			if e != nil {
				t.Fatalf("expr: %s, got:%v, want:%v", c.expr, e.Run(nil), nil)
			}
			continue
		}
		got := e.Run(nil).(float64)
		if got != c.val || expr != c.lastExprNode {
			t.Fatalf("expr: %s, got: %f, %s, want: %f, %s", c.expr, got, expr, c.val, c.lastExprNode)
		}
	}
}

func TestFindSelector(t *testing.T) {
	var cases = []struct {
		expr        string
		field       string
		name        string
		subSelector []string
		found       bool
		last        string
	}{
		{expr: "$", field: "", name: "$", subSelector: nil, found: true, last: ""},
		{expr: "()$", field: "", name: "", subSelector: nil, found: false, last: "()$"},
		{expr: "(0)$", field: "", name: "", subSelector: nil, found: false, last: "(0)$"},
		{expr: "(A0)$", field: "A0", name: "$", subSelector: nil, found: true, last: ""},
		{expr: "(A0)$(A1)$", field: "", name: "", subSelector: nil, found: false, last: "(A0)$(A1)$"},
		{expr: "(A0)$ $(A1)$", field: "A0", name: "$", subSelector: nil, found: true, last: " $(A1)$"},
		{expr: "$a", field: "", name: "", subSelector: nil, found: false, last: "$a"},
		{expr: "$[1]['a']", field: "", name: "$", subSelector: []string{"1", "'a'"}, found: true, last: ""},
		{expr: "$[1][]", field: "", name: "", subSelector: nil, found: false, last: "$[1][]"},
		{expr: "$[[]]", field: "", name: "", subSelector: nil, found: false, last: "$[[]]"},
		{expr: "$[[[]]]", field: "", name: "", subSelector: nil, found: false, last: "$[[[]]]"},
		{expr: "$[(A)$[1]]", field: "", name: "$", subSelector: []string{"(A)$[1]"}, found: true, last: ""},
	}
	for _, c := range cases {
		last := c.expr
		field, name, subSelector, found := findSelector(&last)
		if found != c.found {
			t.Fatalf("%q found: got: %v, want: %v", c.expr, found, c.found)
		}
		if field != c.field {
			t.Fatalf("%q field: got: %q, want: %q", c.expr, field, c.field)
		}
		if name != c.name {
			t.Fatalf("%q name: got: %q, want: %q", c.expr, name, c.name)
		}
		if !reflect.DeepEqual(subSelector, c.subSelector) {
			t.Fatalf("%q subSelector: got: %v, want: %v", c.expr, subSelector, c.subSelector)
		}
		if last != c.last {
			t.Fatalf("%q last: got: %q, want: %q", c.expr, last, c.last)
		}
	}
}
