package tagexpr

import (
	"reflect"
	"testing"
)

func TestVMFunc(t *testing.T) {
	var cases = []struct {
		tagName   string
		structure interface{}
		tests     map[string]interface{}
	}{
		{
			tagName: "tagexpr",
			structure: &struct {
				A int     `tagexpr:"$>0&&$<10"`
				b string  `tagexpr:"{is:$=='test'}{msg:sprintf('want: test, but got: %s',$)}"`
				c float32 `tagexpr:"(A)$+$"`
			}{
				A: 5.0,
				b: "x",
				c: 1,
			},
			tests: map[string]interface{}{
				"A":     true,
				"b.is":  false,
				"b.msg": "want: test, but got: x",
				"c":     6.0,
			},
		},
	}
	for i, c := range cases {
		vm := New(c.tagName)
		// vm.WarmUp(c.structure)
		tagExpr, err := vm.Run(c.structure)
		if err != nil {
			t.Fatal(err)
		}
		for selector, value := range c.tests {
			val := tagExpr.Eval(selector)
			if !reflect.DeepEqual(val, value) {
				t.Fatalf("NO: %d, selector: %q, got: %v, want: %v", i, selector, val, value)
			}
		}
		tagExpr.Range(func(selector string, eval func() interface{}) {
			t.Logf("selector: %s", selector)
			value := c.tests[selector]
			val := eval()
			if !reflect.DeepEqual(val, value) {
				t.Fatalf("NO: %d, selector: %q, got: %v, want: %v", i, selector, val, value)
			}
		})
	}
}
