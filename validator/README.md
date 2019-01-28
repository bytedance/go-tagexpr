# validator [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/bytedance/go-tagexpr/validator)

A powerful validator that supports struct tag expression.

## Example

```go
package validator_test

import (
	"fmt"

	"github.com/bytedance/go-tagexpr/validator"
)

func Example() {
	var vdr = validator.New("vdr")

	type A struct {
		A int `vdr:"$<0||$>=100"`
	}
	a := &A{107}
	fmt.Println(vdr.Validate(a) == nil)

	type B struct {
		B string `vdr:"len($)>1 && regexp('^\\w*$')"`
	}
	b := &B{"abc"}
	fmt.Println(vdr.Validate(b) == nil)

	type C struct {
		C bool `vdr:"{@:(S.A)$>0 && !$}{msg:'C must be false when S.A>0'}"`
		S *A
	}
	c := &C{C: true, S: a}
	fmt.Println(vdr.Validate(c))

	type D struct {
		d []string `vdr:"{@:len($)>0 && $[0]=='D'} {msg:sprintf('Invalid d: %v',$)}"`
	}
	d := &D{d: []string{"x", "y"}}
	fmt.Println(vdr.Validate(d))

	type E struct {
		e map[string]int `vdr:"len($)==$['len']"`
	}
	e := &E{map[string]int{"len": 2}}
	fmt.Println(vdr.Validate(e))

	// Customizes the factory of validation error.
	vdr.SetErrorFactory(func(fieldSelector string) error {
		return fmt.Errorf(`{"succ":false, "error":"invalid parameter: %s"}`, fieldSelector)
	})

	type F struct {
		f struct {
			g int `vdr:"$%3==0"`
		}
	}
	f := &F{}
	f.f.g = 10
	fmt.Println(vdr.Validate(f))

	// Output:
	// true
	// true
	// C must be false when S.A>0
	// Invalid d: [x y]
	// Invalid parameter: e
	// {"succ":false, "error":"invalid parameter: f.g"}
}
```