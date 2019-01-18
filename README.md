# go-tagexpr

An interesting go struct tag expression syntax for field validation, etc.

## Status

In development

## Example (plan)

```go
type T struct {
	A int              `tagexpr:"{match:$<0||$>=100}"`
	B string           `tagexpr:"{match:len($)>1 || regexp('^\\w*$')},{msg:Sprintf('Invalid B:%s',$)}"`
	C bool             `tagexpr:"{match:$(G.J)>0 && $}{msg:'C must be true when T.G.J>0'}"`
	D []string         `tagexpr:"{match:len($)>0 && $[0]=='D'}"`
	E map[string]int   `tagexpr:"{match:$k!='' && $v>0}{match:$['E']>0}"`
	F map[string][]int `tagexpr:"{match:$$v>0 && len($['F'])>0 && $['F'][0]>1}"`
	G struct{ J int }
}
```
