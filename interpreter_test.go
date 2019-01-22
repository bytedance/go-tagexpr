package tagexpr

import (
	"math"
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
		{expr: "!false", val: true},
		{expr: "(false)", val: false},
		{expr: "(!false)", val: true},
		{expr: "(!!false)", val: false},
		{expr: "!!(!false)", val: true},
		{expr: "!(!false)", val: false},
		{expr: "!!(!false)", val: true},
		{expr: "!(false)", val: true},

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
		{expr: "100/(( 2+8)*5 )-(1 +1- 0)", val: 0.0},
		{expr: "1/0", val: math.NaN()},

		{expr: "20%2", val: 0.0},
		{expr: "6 % 5", val: 1.0},
		{expr: "20%(7%5)", val: 0.0},
		{expr: "20%7 %5", val: 1.0},

		{expr: "50 == 5", val: false},
		{expr: "'50'== '50'", val: true},
		{expr: "'50' =='5' == true", val: false},
		{expr: "50== 50 == false", val: false},
		{expr: "50== 50 == true ==true==true", val: true},

		{expr: "50 != 5", val: true},
		{expr: "'50'!= '50'", val: false},
		{expr: "'50' !='5' != true", val: false},
		{expr: "50!= 50 == false", val: true},
		{expr: "50== 50 != true ==true!=true", val: true},

		{expr: "50 > 5", val: true},
		{expr: "50.1 > 50.1", val: false},
		{expr: "3 >12.11", val: false},
		{expr: "3.2 > 2.1", val: true},
		{expr: "'3.2' > '2.1'", val: true},
		{expr: "'13.2' > '2.1'", val: false},

		{expr: "3.2 >= 2.1", val: true},
		{expr: "2.1 >= 2.1", val: true},
		{expr: "2.05 >= 2.1", val: false},
		{expr: "'2.05'>='2.1'", val: false},
		{expr: "'12.05'>='2.1'", val: false},

		{expr: "50 < 5", val: false},
		{expr: "50.1 < 50.1", val: false},
		{expr: "3 <12.11", val: true},
		{expr: "3.2 < 2.1", val: false},
		{expr: "'3.2' < '2.1'", val: false},
		{expr: "'13.2' < '2.1'", val: true},
		{expr: "!('13.2' < '2.1')", val: false},

		{expr: "3.2 <= 2.1", val: false},
		{expr: "2.1 <= 2.1", val: true},
		{expr: "2.05 <= 2.1", val: true},
		{expr: "'2.05'<='2.1'", val: true},
		{expr: "'12.05'<='2.1'", val: true},
	}
	for _, c := range cases {
		t.Log(c.expr)
		vm, err := New(c.expr)
		if err != nil {
			t.Fatal(err)
		}
		val := vm.Run()
		if !reflect.DeepEqual(val, c.val) {
			if f, ok := c.val.(float64); ok && math.IsNaN(f) && math.IsNaN(val.(float64)) {
				continue
			}
			t.Fatalf("expr: %q, got: %v, want: %v", c.expr, val, c.val)
		}
	}
}

func TestSyntaxIncorrect(t *testing.T) {
	var cases = []struct {
		incorrectExpr string
	}{
		{incorrectExpr: "1 + + 'a'"},
	}
	for _, c := range cases {
		_, err := New(c.incorrectExpr)
		if err == nil {
			t.Fatalf("want syntax incorrect: %s", c.incorrectExpr)
		} else {
			t.Log(err)
		}
	}
}
