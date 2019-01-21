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
		{expr: "'true '+('a')", val: "true a"},
		{expr: "'a'+('b'+'c')+'d'", val: "abcd"},
		{expr: "1+(7)+(2)", val: 10.0},
		{expr: "1*2+7+2.2", val: 11.2},
		{expr: "(2*3)+(4*2)", val: 14.0},
		{expr: "1+(2*(3+4))", val: 15.0},
		{expr: "10", val: 10.0},
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
