package validator_test

import (
	"fmt"

	"github.com/bytedance/go-tagexpr/validator"
)

func Example() {
	var vd = validator.New("vd")

	type A struct {
		A int `vd:"$<0||$>=100"`
	}
	a := &A{107}
	fmt.Println(vd.Validate(a) == nil)

	type B struct {
		B string `vd:"len($)>1 && regexp('^\\w*$')"`
	}
	b := &B{"abc"}
	fmt.Println(vd.Validate(b) == nil)

	type C struct {
		C bool `vd:"{@:(S.A)$>0 && !$}{msg:'C must be false when S.A>0'}"`
		S *A
	}
	c := &C{C: true, S: a}
	fmt.Println(vd.Validate(c))

	type D struct {
		d []string `vd:"{@:len($)>0 && $[0]=='D'} {msg:sprintf('Invalid d: %v',$)}"`
	}
	d := &D{d: []string{"x", "y"}}
	fmt.Println(vd.Validate(d))

	type E struct {
		e map[string]int `vd:"len($)==$['len']"`
	}
	e := &E{map[string]int{"len": 2}}
	fmt.Println(vd.Validate(e))

	// Customizes the factory of validation error.
	vd.SetErrorFactory(func(fieldSelector string) error {
		return fmt.Errorf(`{"succ":false, "error":"invalid parameter: %s"}`, fieldSelector)
	})

	type F struct {
		f struct {
			g int `vd:"$%3==0"`
		}
	}
	f := &F{}
	f.f.g = 10
	fmt.Println(vd.Validate(f))

	// Output:
	// true
	// true
	// C must be false when S.A>0
	// Invalid d: [x y]
	// Invalid parameter: e
	// {"succ":false, "error":"invalid parameter: f.g"}
}
