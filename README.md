# go-tagexpr

An interesting go struct tag expression syntax for field validation, etc.

## Status

In development

## Example

```go
type T struct {
	A int              `tagexpr:"$<0||$>=100"`
	B string           `tagexpr:"len($)>1 || regexp('^\\w*$')"`
	C bool             `tagexpr:"{expr1:(G)$['J']>0 && $}{expr2:'C must be true when T.G.J>0'}"`
	D []string         `tagexpr:"{expr1:len($)>0 && $[0]=='D'} {expr2:sprintf('Invalid D:%s',$)}"`
	E map[string]int   `tagexpr:"{expr1:$k!='' && $v>0}{expr2:$['E']>0}"`
	F map[string][]int `tagexpr:"$$v>0 && len($['F'])>0 && $['F'][0]>1"`
	G struct{ J int }
}
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
|`$`|Shorthand for `(X)$`, omit `(X)` to indicate current struct field value|
|`(X)$['A']`|Map or struct field value with name A in the struct field X|
|`(X)$[0]`|The 0th element of the struct field X(type: map, slice, array)|
|`(X)$k`|Traverse each element key of the struct field X(type: map, slice, array)|
|`(X)$v`|Traverse each element value of the struct field X(type: map, slice, array)|
|`len((X)$)`|Built-in function `len`, the length of struct field X|
|`len()`|Built-in function `len`, the length of the current struct field|
|`regexp('^\\w*$', (X)$)`|Regular match the struct field X, return boolean|
|`regexp('^\\w*$')`|Regular match the current struct field, return boolean|
|`sprintf('X value: %v', (X)$)`|`fmt.Sprintf`, format the value of struct field X|

<!-- |`&`|Integer bitwise `and`|
|`\|`|Integer bitwise `or`|
|`^`|Integer bitwise `not` or `xor`|
|`&^`|Integer bitwise `clean`|
|`<<`|Integer bitwise `shift left`|
|`>>`|Integer bitwise `shift right`| -->

Operator priority(high -> low):
* `()` `bool` `string` `float64`
* `*` `/` `%`
* `+` `-`
* `<` `<=` `>` `>=`
* `==` `!=`
* `&&`
* `||`