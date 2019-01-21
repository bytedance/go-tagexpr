package tagexpr

import (
	"reflect"
	"testing"
)

func TestInterpreter(t *testing.T) {
	var cases = []struct {
		expr string
		val  interface{}
	}{
		{expr: "'a'", val: "a"},
		{expr: "('a')", val: "a"},

		{expr: "10", val: 10.0},
		{expr: "(10)", val: 10.0},

		{expr: "true", val: true},
		{expr: "!true", val: false},
		{expr: "!!true", val: true},
		{expr: "false", val: false},
		{expr: "(!false)", val: true},
		// {expr: "!(!false)", val: false},

		{expr: "'true '+('a')", val: "true a"},
		{expr: "'a'+('b'+'c')+'d'", val: "abcd"},

		{expr: "1+(7)+(2)", val: 10.0},
		{expr: "1*2+7+2.2", val: 11.2},
		{expr: "(2*3)+(4*2)", val: 14.0},
		{expr: "1+(2*(3+4))", val: 15.0},

		{expr: "20/2+1+2", val: 13.0},
		{expr: "-20/2+1+2", val: -7.0},
		{expr: "20/2+1-2-1", val: 8.0},
		{expr: "30/(2+1)/5-2-1", val: -1.0},
		{expr: "100/((2+8)*5)-(1+1-0)", val: 0.0},
	}
	for _, c := range cases {
		t.Log(c.expr)
		vm, err := New(c.expr)
		if err != nil {
			t.Fatal(err)
		}
		val := vm.Run()
		if !reflect.DeepEqual(val, c.val) {
			t.Fatalf("expr: %q, got: %v, want: %v", c.expr, val, c.val)
		}
	}
}
