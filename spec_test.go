package tagexpr

import "testing"

func TestReadPairedSymbol(t *testing.T) {
	var cases = []struct {
		expr        string
		val         string
		lastExpr    string
		left, right rune
	}{
		{expr: "'true '+'a'", val: "true ", lastExpr: "+'a'", left: '\'', right: '\''},
		{expr: "((0+1)/(2-1)*9)%2", val: "(0+1)/(2-1)*9", lastExpr: "%2", left: '(', right: ')'},
	}
	for _, c := range cases {
		t.Log(c.expr)
		expr := c.expr
		got := readPairedSymbol(&expr, c.left, c.right)
		if *got != c.val || expr != c.lastExpr {
			t.Fatalf("expr: %q, got: %q, %q, want: %q, %q", c.expr, *got, expr, c.val, c.lastExpr)
		}
	}
}

func TestReadBoolExpr(t *testing.T) {
	var cases = []struct {
		expr     string
		val      bool
		lastExpr string
	}{
		{expr: "false", val: false, lastExpr: ""},
		{expr: "true", val: true, lastExpr: ""},
		{expr: "true ", val: true, lastExpr: " "},
		{expr: "!true&", val: false, lastExpr: "&"},
		{expr: "!false|", val: true, lastExpr: "|"},
		{expr: "!!!!false =", val: !!!!false, lastExpr: " ="},
	}
	for _, c := range cases {
		t.Log(c.expr)
		expr := c.expr
		e := readBoolExpr(&expr)
		got := e.Calculate().(bool)
		if got != c.val || expr != c.lastExpr {
			t.Fatalf("expr: %s, got: %v, %s, want: %v, %s", c.expr, got, expr, c.val, c.lastExpr)
		}
	}
}

func TestReadDigitalExpr(t *testing.T) {
	var cases = []struct {
		expr     string
		val      float64
		lastExpr string
	}{
		{expr: "0.1 +1", val: 0.1, lastExpr: " +1"},
		{expr: "-1\\1", val: -1, lastExpr: "\\1"},
		{expr: "1a", val: 0, lastExpr: ""},
		{expr: "1", val: 1, lastExpr: ""},
		{expr: "1.1", val: 1.1, lastExpr: ""},
		{expr: "1.1/", val: 1.1, lastExpr: "/"},
	}
	for _, c := range cases {
		expr := c.expr
		e := readDigitalExpr(&expr)
		if c.expr == "1a" {
			if e != nil {
				t.Fatalf("expr: %s, got:%v, want:%v", c.expr, e.Calculate(), nil)
			}
			continue
		}
		got := e.Calculate().(float64)
		if got != c.val || expr != c.lastExpr {
			t.Fatalf("expr: %s, got: %f, %s, want: %f, %s", c.expr, got, expr, c.val, c.lastExpr)
		}
	}
}
