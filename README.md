# go-tagexpr

An interesting go struct tag expression syntax for field validation, etc.

## Example

```go
import (
	"github.com/bytedance/go-tagexpr"
)

type T struct {
	A int            `tagexpr:"$<0||$>=100"`
	B string         `tagexpr:"len($)>1 && regexp('^\\w*$')"`
	C bool           `tagexpr:"{expr1:(f.g)$>0 && $}{expr2:'C must be true when T.f.g>0'}"`
	d []string       `tagexpr:"{match:len($)>0 && $[0]=='D'} {msg:sprintf('Invalid d: %v',$)}"`
	e map[string]int `tagexpr:"len($)==$['len']"`
	f struct {
		g int `tagexpr:"$"`
	}
}

vm := tagexpr.New("tagexpr")
err := vm.WarmUp(new(T))
if err != nil {
	panic(err)
}

t := &T{
	A: 107,
	B: "abc",
	C: true,
	d: []string{"x", "y"},
	e: map[string]int{"len": 1},
	f: struct {
		g int `tagexpr:"$"`
	}{1},
}
tagExpr, err := vm.Run(t)
if err != nil {
	panic(err)
}
fmt.Println(tagExpr.Eval("A.$"))
fmt.Println(tagExpr.Eval("B.$"))
fmt.Println(tagExpr.Eval("C.expr1"))
fmt.Println(tagExpr.Eval("C.expr2"))
if !tagExpr.Eval("d.match").(bool) {
	fmt.Println(tagExpr.Eval("d.msg"))
}
fmt.Println(tagExpr.Eval("e.$"))
fmt.Println(tagExpr.Eval("f.g.$"))
// Output:
// true
// true
// true
// C must be true when T.f.g>0
// Invalid d: [x y]
// true
// 1
```

## Syntax

Struct tag syntax spec:

```
type T struct {
    Field1 T1 `tagName:"expression"`
    Field2 T2 `tagName:"{exprName:expression} [{exprName2:expression2}]..."`
    ...
}
```

NOTE: **The `exprName` under the same struct field cannot be the same！**

|Operator or Expression example|Explain|
|-----|---------|
|`true`|bool "true"|
|`false`|bool "false"|
|`1`|float64 "1"|
|`1.0`|float64 "1.0"|
|`'S'`|String "S"|
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
|`(X)$['A']`|Map value with key A in the struct field X|
|`(X)$[0]`|The 0th element of the struct field X(type: map, slice, array)|
|`len((X)$)`|Built-in function `len`, the length of struct field X|
|`len()`|Built-in function `len`, the length of the current struct field|
|`regexp('^\\w*$', (X)$)`|Regular match the struct field X, return boolean|
|`regexp('^\\w*$')`|Regular match the current struct field, return boolean|
|`sprintf('X value: %v', (X)$)`|`fmt.Sprintf`, format the value of struct field X|

<!-- |`(X)$k`|Traverse each element key of the struct field X(type: map, slice, array)|
|`(X)$v`|Traverse each element value of the struct field X(type: map, slice, array)| -->

<!-- |`&`|Integer bitwise `and`|
|`\|`|Integer bitwise `or`|
|`^`|Integer bitwise `not` or `xor`|
|`&^`|Integer bitwise `clean`|
|`<<`|Integer bitwise `shift left`|
|`>>`|Integer bitwise `shift right`| -->

Operator priority(high -> low):
* `()` `bool` `string` `float64` `!`
* `*` `/` `%`
* `+` `-`
* `<` `<=` `>` `>=`
* `==` `!=`
* `&&`
* `||`