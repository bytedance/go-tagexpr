# validator [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/bytedance/go-tagexpr/validator)

A powerful validator that supports struct tag expression.

## Feature

- Support for a variety of common operator
- Support for accessing arrays, slices, members of the dictionary
- Support access to any field in the current structure
- Support access to nested fields, non-exported fields, etc.
- Support registers validator function expression
- Built-in len, sprintf, regexp, email, phone functions
- Support simple mode, or specify error message mode
- Use offset pointers to directly take values, better performance

## Example

```go
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
	info := &InfoRequest{
		Name:   "Alice",
		Age:    18,
		Email:  "henrylee2cn@gmail.com",
		Phone1: "+8618812345678",
		Phone2: "18812345678",
	}
	fmt.Println(vd.Validate(info) == nil)

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

	// Output:
	// true
	// true
	// true
	// C must be false when S.A>0
	// invalid d: [x y]
	// invalid parameter: e
	// {"succ":false, "error":"invalid parameter: f.g"}
}
```

## Syntax

Struct tag syntax spec:

```
type T struct {
	// Simple model
    Field1 T1 `tagName:"expression"`
	// Specify error message mode
    Field2 T2 `tagName:"{@:expression}{msg:expression2}"`
    ...
}
```

|Operator or Operand|Explain|
|-----|---------|
|`!`|not, only valid for boolean value|
|`true` `false`|boolean|
|`0` `0.0`|float64 "0"|
|`''`|String|
|`nil`|nil, undefined|
|`+`|Digital addition or string splicing|
|`-`|Digital subtraction or negative|
|`*`|Digital multiplication|
|`/`|Digital division|
|`%`|division remainder, as: `float64(int64(a)%int64(b))`|
|`==`|`eq`|
|`!=`|`ne`|
|`>`|`gt`|
|`>=`|`ge`|
|`<`|`lt`|
|`<=`|`le`|
|`&&`|Logic `and`|
|`\|\|`|Logic `or`|
|`()`|Expression group|
|`(X)$`|Struct field value named X|
|`(X.Y)$`|Struct field value named X.Y|
|`$`|Shorthand for `(X)$`, omit `(X)` to indicate current struct field value|
|`(X)$['A']`|Map value with key A or struct A sub-field in the struct field X|
|`(X)$[0]`|The 0th element or sub-field of the struct field X(type: map, slice, array, struct)|
|`len((X)$)`|Built-in function `len`, the length of struct field X|
|`regexp('^\\w*$', (X)$)`|Regular match the struct field X, return boolean|
|`regexp('^\\w*$')`|Regular match the current struct field, return boolean|
|`sprintf('X value: %v', (X)$)`|`fmt.Sprintf`, format the value of struct field X|
|`email((X)$)`|Regular match the struct field X, return true if it is email|
|`phone((X)$,<'defaultRegion'>)`|Regular match the struct field X, return true if it is phone|

<!-- |`(X)$k`|Traverse each element key of the struct field X(type: map, slice, array)|
|`(X)$v`|Traverse each element value of the struct field X(type: map, slice, array)| -->

<!-- |`&`|Integer bitwise `and`|
|`\|`|Integer bitwise `or`|
|`^`|Integer bitwise `not` or `xor`|
|`&^`|Integer bitwise `clean`|
|`<<`|Integer bitwise `shift left`|
|`>>`|Integer bitwise `shift right`| -->

Operator priority(high -> low):

* `()` `!` `bool` `float64` `string` `nil`
* `*` `/` `%`
* `+` `-`
* `<` `<=` `>` `>=`
* `==` `!=`
* `&&`
* `||`
