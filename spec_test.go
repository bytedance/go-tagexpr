package tagexpr

import "testing"

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
		got := e.Run().(bool)
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
				t.Fatalf("expr: %s, got:%v, want:%v", c.expr, e.Run(), nil)
			}
			continue
		}
		got := e.Run().(float64)
		if got != c.val || expr != c.lastExprNode {
			t.Fatalf("expr: %s, got: %f, %s, want: %f, %s", c.expr, got, expr, c.val, c.lastExprNode)
		}
	}
}
