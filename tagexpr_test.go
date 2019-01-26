package tagexpr

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"
)

func Example() {
	type T struct {
		A  int             `tagexpr:"$<0||$>=100"`
		B  string          `tagexpr:"len($)>1 && regexp('^\\w*$')"`
		C  bool            `tagexpr:"{expr1:(f.g)$>0 && $}{expr2:'C must be true when T.f.g>0'}"`
		d  []string        `tagexpr:"{@:len($)>0 && $[0]=='D'} {msg:sprintf('Invalid d: %v',$)}"`
		e  map[string]int  `tagexpr:"len($)==$['len']"`
		e2 map[string]*int `tagexpr:"len($)==$['len']"`
		f  struct {
			g int `tagexpr:"$"`
		}
	}

	vm := New("tagexpr")
	err := vm.WarmUp(new(T))
	if err != nil {
		panic(err)
	}

	t := &T{
		A:  107,
		B:  "abc",
		C:  true,
		d:  []string{"x", "y"},
		e:  map[string]int{"len": 1},
		e2: map[string]*int{"len": new(int)},
		f: struct {
			g int `tagexpr:"$"`
		}{1},
	}

	tagExpr, err := vm.Run(t)
	if err != nil {
		panic(err)
	}

	fmt.Println(tagExpr.Eval("A@"))
	fmt.Println(tagExpr.Eval("B@"))
	fmt.Println(tagExpr.Eval("C@expr1"))
	fmt.Println(tagExpr.Eval("C@expr2"))
	if !tagExpr.Eval("d@").(bool) {
		fmt.Println(tagExpr.Eval("d@msg"))
	}
	fmt.Println(tagExpr.Eval("e@"))
	fmt.Println(tagExpr.Eval("e2@"))
	fmt.Println(tagExpr.Eval("f.g@"))

	// Output:
	// true
	// true
	// true
	// C must be true when T.f.g>0
	// Invalid d: [x y]
	// true
	// false
	// 1
}

func BenchmarkTagExpr(b *testing.B) {
	b.StopTimer()
	type T struct {
		a int `bench:"$%3"`
	}
	vm := New("bench")
	err := vm.WarmUp(new(T))
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.StartTimer()
	var t = &T{10}
	for i := 0; i < b.N; i++ {
		tagExpr, err := vm.Run(t)
		if err != nil {
			b.FailNow()
		}
		if tagExpr.EvalFloat("a@") != 1 {
			b.FailNow()
		}
	}
}

func BenchmarkReflect(b *testing.B) {
	b.StopTimer()
	type T struct {
		a int `remainder:"3"`
	}
	b.ReportAllocs()
	b.StartTimer()
	var t = &T{1}
	for i := 0; i < b.N; i++ {
		v := reflect.ValueOf(t).Elem()
		ft, ok := v.Type().FieldByName("a")
		if !ok {
			b.FailNow()
		}
		x, err := strconv.ParseInt(ft.Tag.Get("remainder"), 10, 64)
		if err != nil {
			b.FailNow()
		}
		fv := v.FieldByName("a")
		if fv.Int()%x != 1 {
			b.FailNow()
		}
	}
}

func Test(t *testing.T) {
	g := &struct {
		_ int
		h string `tagexpr:"$"`
		s []string
		m map[string][]string
	}{
		h: "haha",
		s: []string{"1"},
		m: map[string][]string{"0": {"2"}},
	}
	d := "ddd"
	e := new(int)
	*e = 3
	var cases = []struct {
		tagName   string
		structure interface{}
		tests     map[string]interface{}
	}{
		{
			tagName: "tagexpr",
			structure: &struct {
				A  int            `tagexpr:"$>0&&$<10"`
				A2 int            `tagexpr:"{@:$>0&&$<10}"`
				b  string         `tagexpr:"{is:$=='test'}{msg:sprintf('want: test, but got: %s',$)}"`
				c  float32        `tagexpr:"(A)$+$"`
				d  *string        `tagexpr:"$"`
				e  **int          `tagexpr:"$"`
				f  *[3]int        `tagexpr:"{x:len($)}{y:len()}"`
				g  string         `tagexpr:"{x:regexp('g\\d{3}$',$)}{y:regexp('g\\d{3}$')}"`
				h  []string       `tagexpr:"{x:$[1]}{y:$[10]}"`
				i  map[string]int `tagexpr:"{x:$['a']}{y:$[0]}"`
			}{
				A:  5.0,
				A2: 5.0,
				b:  "x",
				c:  1,
				d:  &d,
				e:  &e,
				f:  new([3]int),
				g:  "g123",
				h:  []string{"", "hehe"},
				i:  map[string]int{"a": 7},
			},
			tests: map[string]interface{}{
				"A@":    true,
				"A2@":   true,
				"b@is":  false,
				"b@msg": "want: test, but got: x",
				"c@":    6.0,
				"d@":    d,
				"e@":    float64(*e),
				"f@x":   float64(3),
				"f@y":   float64(3),
				"g@x":   true,
				"g@y":   true,
				"h@x":   "hehe",
				"h@y":   nil,
				"i@x":   7.0,
				"i@y":   nil,
			},
		},
		{
			tagName: "tagexpr",
			structure: &struct {
				A int    `tagexpr:"$>0&&$<10"`
				b string `tagexpr:"{is:$=='test'}{msg:sprintf('want: test, but got: %s',$)}"`
				c struct {
					_ int
					d bool `tagexpr:"$"`
				}
				e *struct {
					_ int
					f bool `tagexpr:"$"`
				}
				g **struct {
					_ int
					h string `tagexpr:"$"`
					s []string
					m map[string][]string
				} `tagexpr:"$['h']"`
				i string `tagexpr:"(g.s)$[0]+(g.m)$['0'][0]==$"`
				j bool   `tagexpr:"!$"`
				k int    `tagexpr:"!$"`
			}{
				A: 5.0,
				b: "x",
				c: struct {
					_ int
					d bool `tagexpr:"$"`
				}{d: true},
				e: &struct {
					_ int
					f bool `tagexpr:"$"`
				}{f: true},
				g: &g,
				i: "12",
			},
			tests: map[string]interface{}{
				"A@":    true,
				"b@is":  false,
				"b@msg": "want: test, but got: x",
				"c.d@":  true,
				"e.f@":  true,
				"g.h@":  "haha",
				"i@":    true,
				"j@":    true,
				"k@":    nil,
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
				t.Fatalf("Eval NO: %d, selector: %q, got: %v, want: %v", i, selector, val, value)
			}
		}
		tagExpr.Range(func(selector string, eval func() interface{}) bool {
			t.Logf("Range selector: %s", selector)
			value := c.tests[selector]
			val := eval()
			if !reflect.DeepEqual(val, value) {
				t.Fatalf("Range NO: %d, selector: %q, got: %v, want: %v", i, selector, val, value)
			}
			return true
		})
	}
}
