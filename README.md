# go-tagexpr

An interesting go struct tag expression syntax for field validation, etc.

## Status

In development

## Example (plan)

```go
type T struct {
	A int              `tagexpr:"{match:$<0||$>=100}"`
	B string           `tagexpr:"{match:len($)>1 || regexp('^\\w*$')},{msg:sprintf('Invalid B:%s',$)}"`
	C bool             `tagexpr:"{match:$(G)['J']>0 && $}{msg:'C must be true when T.G.J>0'}"`
	D []string         `tagexpr:"{match:len($)>0 && $[0]=='D'}"`
	E map[string]int   `tagexpr:"{match:$k!='' && $v>0}{match:$['E']>0}"`
	F map[string][]int `tagexpr:"{match:$$v>0 && len($['F'])>0 && $['F'][0]>1}"`
	G struct{ J int }
}
```

## Syntax

Struct tag syntax spec:

`tagName:"{subtagName:expression} [{subtagName2:expression2}]..."`

|Operator or Expression|Explain|
|-----|---------|
|`+`|Digital addition or string splicing|
|`-`|Digital subtraction|
|`*`|Digital multiplication|
|`/`|Digital division|
|`%`|Digital division remainder|
|`&`|Integer bitwise `and`|
|`\|`|Integer bitwise `or`|
|`^`|Integer bitwise `not` or `xor`|
|`&^`|Integer bitwise `clean`|
|`<<`|Integer bitwise `move left`|
|`>>`|Integer bitwise `move right`|
|`==`|`eq`|
|`!=`|`ne`|
|`>`|`gt`|
|`>=`|`ge`|
|`<`|`lt`|
|`<=`|`le`|
|`&&`|Logic `and`|
|`\|\|`|Logic `or`|
|`()`|Expression group|
|`0`|Digital "0"|
|`'X'`|String "X"|
|`$`|Current struct field|
|`$(X)`|Struct field named X|
|`$['A']`|Struct field or map value with name A in the current struct field|
|`$(X)['A']`|Struct field or map value with name A in the struct field X|
|`$[0]`|The 0th element when the current struct field is a slice or array|
|`$(X)[0]`|The 0th element when the struct field X is a slice or array|
|`$$`|Traverse each element of the current struct field|
|`$(X)$`|Traverse each element of the struct field X|
|`len(X)`|Built-in function `len`, the length of X|
|`sprintf(X)`|`fmt.Sprintf`|
