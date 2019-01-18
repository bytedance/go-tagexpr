# go-tagexpr

An interesting go struct tag expression syntax for field validation, etc.

## Status

In development

## Example (plan)

```go
type T struct {
	A int              `tagx:"{match:$<0||$>=100}"`
	B string           `tagx:"{match:($(G.J)>0 && len($)>1) || regexp('^\\w*$')},{msg:Sprintf('Invalid B:%s',$)}"`
	C bool             `tagx:"{match:!$}{msg:'C must be true'}"`
	D []string         `tagx:"{match:len($)>0 && $[0]=='D'}"`
	E map[string]int   `tagx:"{match:$k!='' && $v>0}{match:$['E']>0}"`
	F map[string][]int `tagx:"{match:$$v>0 && len($['F'])>0 && $['F'][0]>1}"`
	G struct{ J int }
}
```
