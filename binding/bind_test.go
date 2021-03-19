package binding_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/henrylee2cn/ameda"
	"github.com/henrylee2cn/goutil/httpbody"
	"github.com/stretchr/testify/assert"

	"github.com/bytedance/go-tagexpr/v2/binding"
)

func TestRawBody(t *testing.T) {
	type Recv struct {
		S []byte   `raw_body:""`
		F **string `raw_body:"" vd:"@:len($)<3; msg:'f too long'"`
	}
	bodyBytes := []byte("raw_body.............")
	req := newRequest("", nil, nil, bytes.NewReader(bodyBytes))
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.EqualError(t, err, "validating: expr_path=F, cause=f too long")
	assert.Equal(t, bodyBytes, []byte(recv.S))
	bodyCopied, err := binding.GetBody(req)
	assert.NoError(t, err)
	assert.Equal(t, bodyBytes, bodyCopied.Bytes())
	t.Logf("%s", bodyCopied)
}

func TestQueryString(t *testing.T) {
	type metric string
	type count int32

	type Recv struct {
		X **struct {
			A []string  `query:"a"`
			B string    `query:"b"`
			C *[]string `query:"c,required"`
			D *string   `query:"d"`
			E *[]***int `query:"e"`
			F metric    `query:"f"`
			G count     `query:"g"`
		}
		Y string  `query:"y,required"`
		Z *string `query:"z"`
	}
	req := newRequest("http://localhost:8080/?a=a1&a=a2&b=b1&c=c1&c=c2&d=d1&d=d&f=qps&g=1002&e=&e=2&y=y1", nil, nil, nil)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.EqualError(t, err, "binding: expr_path=X.E, cause=parameter type does not match binding data")
	binder.SetLooseZeroMode(true)
	err = binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, 0, ***(*(**recv.X).E)[0])
	assert.Equal(t, 2, ***(*(**recv.X).E)[1])
	assert.Equal(t, []string{"a1", "a2"}, (**recv.X).A)
	assert.Equal(t, "b1", (**recv.X).B)
	assert.Equal(t, []string{"c1", "c2"}, *(**recv.X).C)
	assert.Equal(t, "d1", *(**recv.X).D)
	assert.Equal(t, metric("qps"), (**recv.X).F)
	assert.Equal(t, count(1002), (**recv.X).G)
	assert.Equal(t, "y1", recv.Y)
	assert.Equal(t, (*string)(nil), recv.Z)
}

func TestGetBody(t *testing.T) {
	type Recv struct {
		X **struct {
			E string `json:"e,required" query:"e,required"`
		}
	}
	req := newRequest("http://localhost:8080/", nil, nil, nil)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.EqualError(t, err, "binding: expr_path=X.e, cause=missing required parameter")
}

func TestQueryNum(t *testing.T) {
	type Recv struct {
		X **struct {
			A []int     `query:"a"`
			B int32     `query:"b"`
			C *[]uint16 `query:"c,required"`
			D *float32  `query:"d"`
		}
		Y bool   `query:"y,required"`
		Z *int64 `query:"z"`
	}
	req := newRequest("http://localhost:8080/?a=11&a=12&b=21&c=31&c=32&d=41&d=42&y=true", nil, nil, nil)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, []int{11, 12}, (**recv.X).A)
	assert.Equal(t, int32(21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
	assert.Equal(t, float32(41), *(**recv.X).D)
	assert.Equal(t, true, recv.Y)
	assert.Equal(t, (*int64)(nil), recv.Z)
}

func TestHeaderString(t *testing.T) {
	type Recv struct {
		X **struct {
			A []string  `header:"X-A"`
			B string    `header:"X-B"`
			C *[]string `header:"X-C,required"`
			D *string   `header:"X-D"`
		}
		Y string  `header:"X-Y,required"`
		Z *string `header:"X-Z"`
	}
	header := make(http.Header)
	header.Add("X-A", "a1")
	header.Add("X-A", "a2")
	header.Add("X-B", "b1")
	header.Add("X-C", "c1")
	header.Add("X-C", "c2")
	header.Add("X-D", "d1")
	header.Add("X-D", "d2")
	header.Add("X-Y", "y1")
	req := newRequest("", header, nil, nil)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a1", "a2"}, (**recv.X).A)
	assert.Equal(t, "b1", (**recv.X).B)
	assert.Equal(t, []string{"c1", "c2"}, *(**recv.X).C)
	assert.Equal(t, "d1", *(**recv.X).D)
	assert.Equal(t, "y1", recv.Y)
	assert.Equal(t, (*string)(nil), recv.Z)
}

func TestHeaderNum(t *testing.T) {
	type Recv struct {
		X **struct {
			A []int     `header:"X-A"`
			B int32     `header:"X-B"`
			C *[]uint16 `header:"X-C,required"`
			D *float32  `header:"X-D"`
		}
		Y bool   `header:"X-Y,required"`
		Z *int64 `header:"X-Z"`
	}
	header := make(http.Header)
	header.Add("X-A", "11")
	header.Add("X-A", "12")
	header.Add("X-B", "21")
	header.Add("X-C", "31")
	header.Add("X-C", "32")
	header.Add("X-D", "41")
	header.Add("X-D", "42")
	header.Add("X-Y", "true")
	req := newRequest("", header, nil, nil)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, []int{11, 12}, (**recv.X).A)
	assert.Equal(t, int32(21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
	assert.Equal(t, float32(41), *(**recv.X).D)
	assert.Equal(t, true, recv.Y)
	assert.Equal(t, (*int64)(nil), recv.Z)
}

func TestCookieString(t *testing.T) {
	type Recv struct {
		X **struct {
			A []string  `cookie:"a"`
			B string    `cookie:"b"`
			C *[]string `cookie:"c,required"`
			D *string   `cookie:"d"`
		}
		Y string  `cookie:"y,required"`
		Z *string `cookie:"z"`
	}
	cookies := []*http.Cookie{
		{Name: "a", Value: "a1"},
		{Name: "a", Value: "a2"},
		{Name: "b", Value: "b1"},
		{Name: "c", Value: "c1"},
		{Name: "c", Value: "c2"},
		{Name: "d", Value: "d1"},
		{Name: "d", Value: "d2"},
		{Name: "y", Value: "y1"},
	}
	req := newRequest("", nil, cookies, nil)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, []string{"a1", "a2"}, (**recv.X).A)
	assert.Equal(t, "b1", (**recv.X).B)
	assert.Equal(t, []string{"c1", "c2"}, *(**recv.X).C)
	assert.Equal(t, "d1", *(**recv.X).D)
	assert.Equal(t, "y1", recv.Y)
	assert.Equal(t, (*string)(nil), recv.Z)
}

func TestCookieNum(t *testing.T) {
	type Recv struct {
		X **struct {
			A []int     `cookie:"a"`
			B int32     `cookie:"b"`
			C *[]uint16 `cookie:"c,required"`
			D *float32  `cookie:"d"`
		}
		Y bool   `cookie:"y,required"`
		Z *int64 `cookie:"z"`
	}
	cookies := []*http.Cookie{
		{Name: "a", Value: "11"},
		{Name: "a", Value: "12"},
		{Name: "b", Value: "21"},
		{Name: "c", Value: "31"},
		{Name: "c", Value: "32"},
		{Name: "d", Value: "41"},
		{Name: "d", Value: "42"},
		{Name: "y", Value: "t"},
	}
	req := newRequest("", nil, cookies, nil)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, []int{11, 12}, (**recv.X).A)
	assert.Equal(t, int32(21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
	assert.Equal(t, float32(41), *(**recv.X).D)
	assert.Equal(t, true, recv.Y)
	assert.Equal(t, (*int64)(nil), recv.Z)
}

func TestFormString(t *testing.T) {
	type Recv struct {
		X **struct {
			A []string  `form:"a"`
			B string    `form:"b"`
			C *[]string `form:"c,required"`
			D *string   `form:"d"`
		}
		Y string  `form:"y,required"`
		Z *string `form:"z"`
	}
	values := make(url.Values)
	values.Add("a", "a1")
	values.Add("a", "a2")
	values.Add("b", "b1")
	values.Add("c", "c1")
	values.Add("c", "c2")
	values.Add("d", "d1")
	values.Add("d", "d2")
	values.Add("y", "y1")
	for _, f := range []httpbody.Files{nil, {
		"f1": []httpbody.File{
			httpbody.NewFile("txt", strings.NewReader("f11 text.")),
		},
	}} {
		contentType, bodyReader := httpbody.NewFormBody2(values, f)
		header := make(http.Header)
		header.Set("Content-Type", contentType)
		req := newRequest("", header, nil, bodyReader)
		recv := new(Recv)
		binder := binding.New(nil)
		err := binder.BindAndValidate(recv, req, nil)
		assert.NoError(t, err)
		assert.Equal(t, []string{"a1", "a2"}, (**recv.X).A)
		assert.Equal(t, "b1", (**recv.X).B)
		assert.Equal(t, []string{"c1", "c2"}, *(**recv.X).C)
		assert.Equal(t, "d1", *(**recv.X).D)
		assert.Equal(t, "y1", recv.Y)
		assert.Equal(t, (*string)(nil), recv.Z)
	}
}

func TestFormNum(t *testing.T) {
	type Recv struct {
		X **struct {
			A []int     `form:"a"`
			B int32     `form:"b"`
			C *[]uint16 `form:"c,required"`
			D *float32  `form:"d"`
		}
		Y bool   `form:"y,required"`
		Z *int64 `form:"z"`
	}
	values := make(url.Values)
	values.Add("a", "11")
	values.Add("a", "12")
	values.Add("b", "-21")
	values.Add("c", "31")
	values.Add("c", "32")
	values.Add("d", "41")
	values.Add("d", "42")
	values.Add("y", "1")
	for _, f := range []httpbody.Files{nil, {
		"f1": []httpbody.File{
			httpbody.NewFile("txt", strings.NewReader("f11 text.")),
		},
	}} {
		contentType, bodyReader := httpbody.NewFormBody2(values, f)
		header := make(http.Header)
		header.Set("Content-Type", contentType)
		req := newRequest("", header, nil, bodyReader)
		recv := new(Recv)
		binder := binding.New(nil)
		err := binder.BindAndValidate(recv, req, nil)
		assert.NoError(t, err)
		assert.Equal(t, []int{11, 12}, (**recv.X).A)
		assert.Equal(t, int32(-21), (**recv.X).B)
		assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
		assert.Equal(t, float32(41), *(**recv.X).D)
		assert.Equal(t, true, recv.Y)
		assert.Equal(t, (*int64)(nil), recv.Z)
	}
}

func TestJSON(t *testing.T) {
	// binding.ResetJSONUnmarshaler(false, json.Unmarshal)
	type metric string
	type count int32
	type ZS struct {
		Z *int64
	}
	type Recv struct {
		X **struct {
			A []string          `json:"a"`
			B int32             `json:""`
			C *[]uint16         `json:",required"`
			D *float32          `json:"d"`
			E metric            `json:"e"`
			F count             `json:"f"`
			M map[string]string `json:"m"`
		}
		Y string `json:"y,required"`
		ZS
	}

	bodyReader := strings.NewReader(`{
		"X": {
			"a": ["a1","a2"],
			"B": 21,
			"C": [31,32],
			"d": 41,
			"e": "qps",
			"f": 100,
			"m": {"a":"x"}
		},
		"Z": 6
	}`)

	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	req := newRequest("", header, nil, bodyReader)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.Error(t, err)
	assert.Equal(t, &binding.Error{ErrType: "binding", FailField: "y", Msg: "missing required parameter"}, err)
	assert.Equal(t, []string{"a1", "a2"}, (**recv.X).A)
	assert.Equal(t, int32(21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
	assert.Equal(t, float32(41), *(**recv.X).D)
	assert.Equal(t, metric("qps"), (**recv.X).E)
	assert.Equal(t, count(100), (**recv.X).F)
	assert.Equal(t, map[string]string{"a": "x"}, (**recv.X).M)
	assert.Equal(t, "", recv.Y)
	assert.Equal(t, (int64)(6), *recv.Z)
}

func TestNonstruct(t *testing.T) {
	bodyReader := strings.NewReader(`{
		"X": {
			"a": ["a1","a2"],
			"B": 21,
			"C": [31,32],
			"d": 41,
			"e": "qps",
			"f": 100
		},
		"Z": 6
	}`)

	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	req := newRequest("", header, nil, bodyReader)
	var recv interface{}
	binder := binding.New(nil)
	err := binder.BindAndValidate(&recv, req, nil)
	assert.NoError(t, err)
	b, err := json.Marshal(recv)
	assert.NoError(t, err)
	t.Logf("%s", b)

	bodyReader = strings.NewReader("b=334ddddd&token=yoMba34uspjVQEbhflgTRe2ceeDFUK32&type=url_verification")
	header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	req = newRequest("", header, nil, bodyReader)
	recv = nil
	err = binder.BindAndValidate(&recv, req, nil)
	assert.NoError(t, err)
	b, err = json.Marshal(recv)
	assert.NoError(t, err)
	t.Logf("%s", b)
}

func BenchmarkBindJSON(b *testing.B) {
	type Recv struct {
		X **struct {
			A []string `json:"a"`
			B int32
			C *[]uint16
			D *float32 `json:"d"`
		}
		Y string `json:"y"`
	}
	binder := binding.New(nil)
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	test := func() {
		bodyReader := strings.NewReader(`{
		"X": {
			"a": ["a1","a2"],
			"B": 21,
			"C": [31,32],
			"d": 41
		},
		"y": "y1"
	}`)
		req := newRequest("", header, nil, bodyReader)
		recv := new(Recv)
		err := binder.Bind(recv, req, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
	test()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		test()
	}
}

func BenchmarkStdJSON(b *testing.B) {
	type Recv struct {
		X **struct {
			A []string `json:"a"`
			B int32
			C *[]uint16
			D *float32 `json:"d"`
		}
		Y string `json:"y"`
	}
	header := make(http.Header)
	header.Set("Content-Type", "application/json")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bodyReader := strings.NewReader(`{
			"X": {
				"a": ["a1","a2"],
				"B": 21,
				"C": [31,32],
				"d": 41
			},
			"y": "y1"
		}`)

		req := newRequest("", header, nil, bodyReader)
		recv := new(Recv)
		body, err := ioutil.ReadAll(req.Body)
		req.Body.Close()
		if err != nil {
			b.Fatal(err)
		}
		err = json.Unmarshal(body, recv)
		if err != nil {
			b.Fatal(err)
		}
	}
}

type testPathParams struct{}

func (testPathParams) Get(name string) (string, bool) {
	switch name {
	case "a":
		return "a1", true
	case "b":
		return "-21", true
	case "c":
		return "31", true
	case "d":
		return "41", true
	case "y":
		return "y1", true
	case "name":
		return "henrylee2cn", true
	default:
		return "", false
	}
}

func TestPath(t *testing.T) {
	type Recv struct {
		X **struct {
			A []string  `path:"a"`
			B int32     `path:"b"`
			C *[]uint16 `path:"c,required"`
			D *float32  `path:"d"`
		}
		Y string `path:"y,required"`
		Z *int64
	}

	req := newRequest("", nil, nil, nil)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, new(testPathParams))
	assert.NoError(t, err)
	assert.Equal(t, []string{"a1"}, (**recv.X).A)
	assert.Equal(t, int32(-21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31}, (**recv.X).C)
	assert.Equal(t, float32(41), *(**recv.X).D)
	assert.Equal(t, "y1", recv.Y)
	assert.Equal(t, (*int64)(nil), recv.Z)
}

type testPathParams2 struct{}

func (testPathParams2) Get(name string) (string, bool) {
	switch name {
	case "e":
		return "123", true
	default:
		return "", false
	}
}

func TestDefault(t *testing.T) {
	type S struct {
		SS string `json:"ss"`
	}

	type Recv struct {
		X **struct {
			A          []string           `path:"a" json:"a"`
			B          int32              `path:"b" default:"32"`
			C          bool               `json:"c" default:"true"`
			D          *float32           `default:"123.4"`
			E          *[]string          `default:"['a','b','c','d,e,f']"`
			F          map[string]string  `default:"{'a':'\"\\'1','\"b':'c','c':'2'}"`
			G          map[string]int64   `default:"{'a':1,'b':2,'c':3}"`
			H          map[string]float64 `default:"{'a':0.1,'b':1.2,'c':2.3}"`
			I          map[string]float64 `default:"{'\"a\"':0.1,'b':1.2,'c':2.3}"`
			Empty      string             `default:""`
			Null       string             `default:""`
			CommaSpace string             `default:",a:c "`
			Dash       string             `default:"-"`
			// InvalidInt int                `default:"abc"`
			// InvalidMap map[string]string  `default:"abc"`
		}
		Y       string `json:"y" default:"y1"`
		Z       int64
		W       string                          `json:"w"`
		V       []int64                         `json:"u" default:"[1,2,3]"`
		U       []float32                       `json:"u" default:"[1.1,2,3]"`
		T       *string                         `json:"t" default:"t1"`
		S       S                               `default:"{'ss':'test'}"`
		O       *S                              `default:"{'ss':'test2'}"`
		Complex map[string][]map[string][]int64 `default:"{'a':[{'aa':[1,2,3], 'bb':[4,5]}],'b':[{}]}"`
	}

	bodyReader := strings.NewReader(`{
		"X": {
			"a": ["a1","a2"]
		},
		"Z": 6
	}`)

	// var nilMap map[string]string
	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	req := newRequest("", header, nil, bodyReader)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, new(testPathParams2))
	assert.NoError(t, err)
	assert.Equal(t, []string{"a1", "a2"}, (**recv.X).A)
	assert.Equal(t, int32(32), (**recv.X).B)
	assert.Equal(t, true, (**recv.X).C)
	assert.Equal(t, float32(123.4), *(**recv.X).D)
	assert.Equal(t, []string{"a", "b", "c", "d,e,f"}, *(**recv.X).E)
	assert.Equal(t, map[string]string{"a": "\"'1", "\"b": "c", "c": "2"}, (**recv.X).F)
	assert.Equal(t, map[string]int64{"a": 1, "b": 2, "c": 3}, (**recv.X).G)
	assert.Equal(t, map[string]float64{"a": 0.1, "b": 1.2, "c": 2.3}, (**recv.X).H)
	assert.Equal(t, map[string]float64{"\"a\"": 0.1, "b": 1.2, "c": 2.3}, (**recv.X).I)
	assert.Equal(t, "", (**recv.X).Empty)
	assert.Equal(t, "", (**recv.X).Null)
	assert.Equal(t, ",a:c ", (**recv.X).CommaSpace)
	assert.Equal(t, "-", (**recv.X).Dash)
	// assert.Equal(t, 0, (**recv.X).InvalidInt)
	// assert.Equal(t, nilMap, (**recv.X).InvalidMap)
	assert.Equal(t, "y1", recv.Y)
	assert.Equal(t, "t1", *recv.T)
	assert.Equal(t, int64(6), recv.Z)
	assert.Equal(t, []int64{1, 2, 3}, recv.V)
	assert.Equal(t, []float32{1.1, 2, 3}, recv.U)
	assert.Equal(t, S{SS: "test"}, recv.S)
	assert.Equal(t, &S{SS: "test2"}, recv.O)
	assert.Equal(t, map[string][]map[string][]int64{"a": {{"aa": {1, 2, 3}, "bb": []int64{4, 5}}}, "b": {map[string][]int64{}}}, recv.Complex)
}

func TestTimeLayout(t *testing.T) {
	type alias time.Time

	type Recv struct {
		X struct {
			A  time.Time    `path:"a" layout:"2006-01-02"`
			B  time.Time    `cookie:"b"` // using default layout
			C  time.Time    `query:"c" layout:"2006-01-02"`
			D  time.Time    `form:"d" layout:"2006-01-02"`
			E  time.Time    `cookie:"e" layout:"2006-01-02"`
			F  time.Time    `header:"F" layout:"2006-01-02"`
			G  time.Time    `query:"g" layout:"2006-01-02"`
			H  time.Time    `cookie:"h" layout:"2006-01-02"`
			I  time.Time    `query:"i" layout:"2006-01-02" default:"2020-03-03"`
			J  []time.Time  `query:"j" layout:"2006-01-02"`
			K  []*time.Time `cookie:"k" layout:"2006-01-02"`
			L  alias        `form:"L" layout:"2006-01-02" default:"2020-03-03"`
			II alias        `form:"I" layout:"2006-01-02" default:"2020-03-03"`
		}
		Z time.Time  `layout:"2006-01-02"` // auto binding
		Y *time.Time `layout:"2006-01-02"` // auto binding
	}

	form := make(url.Values)
	form.Add("d", "2020-03-03")
	form.Add("Y", "2020-03-03")
	form.Add("Z", "2020-03-03")
	form.Add("L", "2020-03-03")

	header := make(http.Header)
	contentType, bodyReader := httpbody.NewFormBody2(form, nil)
	header.Set("Content-Type", contentType)
	header.Set("F", "2020-03-03")

	req := newRequest("http://localhost?c=2020-03-03&j=2020-03-03&j=2020-03-04", header, []*http.Cookie{
		{Name: "b", Value: "Mon, 03 Mar 2020 00:00:00 UTC"},
		{Name: "e", Value: "2020-03-03"},
		{Name: "h", Value: "20200303"},
		{Name: "k", Value: "2020-03-03"},
		{Name: "k", Value: "2020-03-04"},
	}, bodyReader)
	recv := new(Recv)
	binder := binding.New(nil)

	ts1, _ := time.Parse("2006-01-02", "2020-03-03")
	ts2, _ := time.Parse("2006-01-02", "2020-03-04")
	err := binder.BindAndValidate(recv, req, new(testPathParams2))
	assert.NoError(t, err)
	assert.Equal(t, ts1, recv.X.B)
	assert.Equal(t, ts1, recv.X.C)
	assert.Equal(t, ts1, recv.X.D)
	assert.Equal(t, ts1, recv.X.E)
	assert.Equal(t, ts1, recv.X.F)
	assert.Equal(t, time.Time{}, recv.X.G) // not assigned value
	assert.Equal(t, time.Time{}, recv.X.H) // invalid time value
	assert.Equal(t, ts1, recv.X.I)
	assert.Equal(t, alias(ts1), recv.X.II)
	assert.Equal(t, []time.Time{ts1, ts2}, recv.X.J)
	assert.Equal(t, []*time.Time{&ts1, &ts2}, recv.X.K)
	assert.Equal(t, alias(ts1), recv.X.L)
	assert.Equal(t, ts1, recv.Z)
	assert.Equal(t, ts1, *recv.Y)
}

func TestTimeLayout_RawBody(t *testing.T) {
	type alias time.Time

	type Recv struct {
		X struct {
			A time.Time `raw_body:"a" layout:"2006-01-02"`
		}
		Z time.Time `raw_body:"z" layout:"2006-01-02"`
		Y alias     `raw_body:"y" layout:"2006-01-02"`
	}

	header := make(http.Header)
	bodyBytes := []byte("2020-03-03")
	req := newRequest("http://localhost", header, nil, bytes.NewReader(bodyBytes))
	recv := new(Recv)
	binder := binding.New(nil)

	ts, _ := time.Parse("2006-01-02", "2020-03-03")
	err := binder.BindAndValidate(recv, req, new(testPathParams2))
	assert.NoError(t, err)
	assert.Equal(t, ts, recv.X.A)
	assert.Equal(t, ts, recv.Z)
	assert.Equal(t, alias(ts), recv.Y)
}

func TestAuto(t *testing.T) {
	type Recv struct {
		A string `vd:"$!=''"`
		B string
		C string
		D string `query:"D,required" form:"D,required"`
		E string `cookie:"e" json:"e"`
	}
	query := make(url.Values)
	query.Add("A", "a")
	query.Add("B", "b")
	query.Add("C", "c")
	query.Add("D", "d-from-query")
	contentType, bodyReader, err := httpbody.NewJSONBody(map[string]string{"e": "e-from-jsonbody"})
	assert.NoError(t, err)
	header := make(http.Header)
	header.Set("Content-Type", contentType)
	req := newRequest("http://localhost/?"+query.Encode(), header, []*http.Cookie{
		{Name: "e", Value: "e-from-cookie"},
	}, bodyReader)
	recv := new(Recv)
	binder := binding.New(nil)
	err = binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, "a", recv.A)
	assert.Equal(t, "b", recv.B)
	assert.Equal(t, "c", recv.C)
	assert.Equal(t, "d-from-query", recv.D)
	assert.Equal(t, "e-from-cookie", recv.E)

	query = make(url.Values)
	query.Add("D", "d-from-query")
	form := make(url.Values)
	form.Add("B", "b")
	form.Add("C", "c")
	form.Add("D", "d-from-form")
	contentType, bodyReader = httpbody.NewFormBody2(form, nil)
	header = make(http.Header)
	header.Set("Content-Type", contentType)
	req = newRequest("http://localhost/?"+query.Encode(), header, nil, bodyReader)
	recv = new(Recv)
	err = binder.Bind(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, "", recv.A)
	assert.Equal(t, "b", recv.B)
	assert.Equal(t, "c", recv.C)
	assert.Equal(t, "d-from-form", recv.D)
	err = binder.Validate(recv)
	assert.EqualError(t, err, "validating: expr_path=A, cause=invalid")
}

func TestTypeUnmarshal(t *testing.T) {
	type Recv struct {
		A time.Time   `form:"t1"`
		B *time.Time  `query:"t2"`
		C []time.Time `query:"t2"`
	}
	query := make(url.Values)
	query.Add("t2", "2019-09-04T14:05:24+08:00")
	query.Add("t2", "2019-09-04T18:05:24+08:00")
	form := make(url.Values)
	form.Add("t1", "2019-09-03T18:05:24+08:00")
	contentType, bodyReader := httpbody.NewFormBody2(form, nil)
	header := make(http.Header)
	header.Set("Content-Type", contentType)
	req := newRequest("http://localhost/?"+query.Encode(), header, nil, bodyReader)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	t1, err := time.Parse(time.RFC3339, "2019-09-03T18:05:24+08:00")
	assert.NoError(t, err)
	assert.Equal(t, t1, recv.A)
	t21, err := time.Parse(time.RFC3339, "2019-09-04T14:05:24+08:00")
	assert.NoError(t, err)
	assert.Equal(t, t21, *recv.B)
	t22, err := time.Parse(time.RFC3339, "2019-09-04T18:05:24+08:00")
	assert.NoError(t, err)
	assert.Equal(t, []time.Time{t21, t22}, recv.C)
	t.Logf("%v", recv)
}

func TestOption(t *testing.T) {
	type Recv struct {
		X *struct {
			C int `json:"c,required"`
			D int `json:"d"`
		} `json:"X"`
		Y string `json:"y"`
	}
	header := make(http.Header)
	header.Set("Content-Type", "application/json")

	bodyReader := strings.NewReader(`{
			"X": {
				"c": 21,
				"d": 41
			},
			"y": "y1"
		}`)
	req := newRequest("", header, nil, bodyReader)
	recv := new(Recv)
	binder := binding.New(nil)
	err := binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, 21, recv.X.C)
	assert.Equal(t, 41, recv.X.D)
	assert.Equal(t, "y1", recv.Y)

	bodyReader = strings.NewReader(`{
			"X": {
			},
			"y": "y1"
		}`)
	req = newRequest("", header, nil, bodyReader)
	recv = new(Recv)
	binder = binding.New(nil)
	err = binder.BindAndValidate(recv, req, nil)
	assert.EqualError(t, err, "binding: expr_path=X.c, cause=missing required parameter")
	assert.Equal(t, 0, recv.X.C)
	assert.Equal(t, 0, recv.X.D)
	assert.Equal(t, "y1", recv.Y)

	bodyReader = strings.NewReader(`{
			"y": "y1"
		}`)
	req = newRequest("", header, nil, bodyReader)
	recv = new(Recv)
	binder = binding.New(nil)
	err = binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.True(t, recv.X == nil)
	assert.Equal(t, "y1", recv.Y)

	type Recv2 struct {
		X *struct {
			C int `json:"c,required"`
			D int `json:"d"`
		} `json:"X,required"`
		Y string `json:"y"`
	}
	bodyReader = strings.NewReader(`{
			"y": "y1"
		}`)
	req = newRequest("", header, nil, bodyReader)
	recv2 := new(Recv2)
	binder = binding.New(nil)
	err = binder.BindAndValidate(recv2, req, nil)
	assert.EqualError(t, err, "binding: expr_path=X, cause=missing required parameter")
	assert.True(t, recv2.X == nil)
	assert.Equal(t, "y1", recv2.Y)
}

func newRequest(u string, header http.Header, cookies []*http.Cookie, bodyReader io.Reader) *http.Request {
	if header == nil {
		header = make(http.Header)
	}
	var method = "GET"
	var body io.ReadCloser
	if bodyReader != nil {
		method = "POST"
		body = ioutil.NopCloser(bodyReader)
	}
	if u == "" {
		u = "http://localhost"
	}
	urlObj, _ := url.Parse(u)
	req := &http.Request{
		Method: method,
		URL:    urlObj,
		Body:   body,
		Header: header,
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

func TestQueryStringIssue(t *testing.T) {
	type Timestamp struct {
		time.Time
	}
	type Recv struct {
		Name *string    `query:"name"`
		T    *Timestamp `query:"t"`
	}
	req := newRequest("http://localhost:8080/?name=test", nil, nil, nil)
	recv := new(Recv)
	binder := binding.New(nil)
	binder.SetLooseZeroMode(true)
	err := binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, ameda.StringToStringPtr("test"), recv.Name)
	assert.Equal(t, (*Timestamp)(nil), recv.T)
}

func TestQueryTypes(t *testing.T) {
	type metric string
	type count int32
	type metrics []string
	type filter struct {
		Col1 string
	}

	type Recv struct {
		A metric `vd:"$!=''"`
		B count
		C *count
		D metrics `query:"D,required" form:"D,required"`
		E metric  `cookie:"e" json:"e"`
		F filter  `json:"f"`
	}
	query := make(url.Values)
	query.Add("A", "qps")
	query.Add("B", "123")
	query.Add("C", "321")
	query.Add("D", "dau")
	query.Add("D", "dnu")
	contentType, bodyReader, err := httpbody.NewJSONBody(
		map[string]interface{}{
			"e": "e-from-jsonbody",
			"f": filter{Col1: "abc"},
		},
	)
	assert.NoError(t, err)
	header := make(http.Header)
	header.Set("Content-Type", contentType)
	req := newRequest("http://localhost/?"+query.Encode(), header, []*http.Cookie{
		{Name: "e", Value: "e-from-cookie"},
	}, bodyReader)
	recv := new(Recv)
	binder := binding.New(nil)
	err = binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, metric("qps"), recv.A)
	assert.Equal(t, count(123), recv.B)
	assert.Equal(t, count(321), *recv.C)
	assert.Equal(t, metrics{"dau", "dnu"}, recv.D)
	assert.Equal(t, metric("e-from-cookie"), recv.E)
	assert.Equal(t, filter{Col1: "abc"}, recv.F)
}

func TestNoTagIssue(t *testing.T) {
	type x int
	type T struct {
		x
		x2 x
		a  int
		B  int
	}
	req := newRequest("http://localhost:8080/?x=11&x2=12&a=1&B=2", nil, nil, nil)
	recv := new(T)
	binder := binding.New(nil)
	binder.SetLooseZeroMode(true)
	err := binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, x(0), recv.x)
	assert.Equal(t, x(0), recv.x2)
	assert.Equal(t, 0, recv.a)
	assert.Equal(t, 2, recv.B)
}

func TestRegTypeUnmarshal(t *testing.T) {
	type Q struct {
		A int
		B string
	}
	type T struct {
		Q Q `query:"q"`
	}
	err := binding.RegTypeUnmarshal(reflect.TypeOf(Q{}), func(v string, emptyAsZero bool) (reflect.Value, error) {
		var q Q
		if len(v) == 0 {
			return reflect.ValueOf(q), nil
		}
		err := json.Unmarshal([]byte(v), &q)
		if err != nil {
			return reflect.Value{}, err
		}
		return reflect.ValueOf(q), nil
	})
	assert.NoError(t, err)
	var values = url.Values{}
	b, err := json.Marshal(Q{A: 2, B: "y"})
	assert.NoError(t, err)
	values.Add("q", string(b))
	req := newRequest("http://localhost:8080/?"+values.Encode(), nil, nil, nil)
	recv := new(T)
	binder := binding.New(nil)
	err = binder.BindAndValidate(recv, req, nil)
	assert.NoError(t, err)
	assert.Equal(t, 2, recv.Q.A)
	assert.Equal(t, "y", recv.Q.B)
}
