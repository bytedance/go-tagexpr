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
		{expr: "'true '+'a'", val: "true a"},
		{expr: "'a'+'b'+'c'", val: "abc"},
		{expr: "1+7+2", val: 10.0},
		{expr: "1+7+2.2", val: 10.2},
		// {expr: "((0+1)/(2-1)*9)%2", val: 1},
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
