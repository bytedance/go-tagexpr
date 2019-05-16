package validator_test

import (
	"fmt"

	"github.com/bytedance/go-tagexpr/validator"
)

func Example() {
	var vd = validator.New("vd")

	type InfoRequest struct {
		Name   string `vd:"($!='Alice'||(Age)$==18) && regexp('\\w')"`
		Age    int    `vd:"$>0"`
		Email  string `vd:"email($)"`
		Phone1 string `vd:"phone($)"`
		Phone2 string `vd:"phone($,'CN')"`
	}
	info := InfoRequest{
		Name:   "Alice",
		Age:    18,
		Email:  "henrylee2cn@gmail.com",
		Phone1: "+8618812345678",
		Phone2: "18812345678",
	}
	fmt.Println(vd.Validate(info) == nil)

	type A struct {
		A    int `vd:"$<0||$>=100"`
		Info interface{}
	}
	info.Email = "xxx"
	a := &A{A: 107, Info: info}
	fmt.Println(vd.Validate(a))

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
		d []string `vd:"{@:len($)>0 && $[0]=='D'} {msg:sprintf('invalid d: %v',$)}"`
	}
	d := &D{d: []string{"x", "y"}}
	fmt.Println(vd.Validate(d))

	type E struct {
		e map[string]int `vd:"len($)==$['len']"`
	}
	e := &E{map[string]int{"len": 2}}
	fmt.Println(vd.Validate(e))

	// Customizes the factory of validation error.
	vd.SetErrorFactory(func(fieldSelector, msg string) error {
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

	fmt.Println(vd.Validate(map[string]*F{"a": f}))

	fmt.Println(vd.Validate([][1]*F{{f}}))

	f = nil
	fmt.Println(vd.Validate(f))

	fmt.Println(vd.Validate(map[string]*F{}))

	fmt.Println(vd.Validate([]*F{}))

	// Output:
	// true
	// invalid parameter: Email
	// true
	// C must be false when S.A>0
	// invalid d: [x y]
	// invalid parameter: e
	// {"succ":false, "error":"invalid parameter: f.g"}
	// {"succ":false, "error":"invalid parameter: f.g"}
	// {"succ":false, "error":"invalid parameter: f.g"}
	// cannot run nil data
	// <nil>
	// <nil>
}
