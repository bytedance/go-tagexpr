package binding_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bytedance/go-tagexpr/binding"
	"github.com/henrylee2cn/goutil/httpbody"
	"github.com/stretchr/testify/assert"
)

func TestRawBody(t *testing.T) {
	type Recv struct {
		rawBody **struct {
			A []byte   `api:"{raw_body:nil}"`
			B *[]byte  `api:"{raw_body:nil}"`
			C **[]byte `api:"{raw_body:nil}"`
			D string   `api:"{raw_body:nil}"`
			E *string  `api:"{raw_body:nil}"`
			F **string `api:"{raw_body:nil}{@:len($)<3}{msg:'too long'}"`
		}
		S string `api:"{raw_body:nil}"`
	}
	bodyBytes := []byte("rawbody.............")
	req := newRequest("", nil, nil, bytes.NewReader(bodyBytes))
	recv := new(Recv)
	binder := binding.New("api")
	err := binder.BindAndValidate(recv, req, nil)
	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "too long")
	for _, v := range []interface{}{
		(**recv.rawBody).A,
		*(**recv.rawBody).B,
		**(**recv.rawBody).C,
		[]byte((**recv.rawBody).D),
		[]byte(*(**recv.rawBody).E),
		[]byte(**(**recv.rawBody).F),
		[]byte(recv.S),
	} {
		assert.Equal(t, bodyBytes, v)
	}
}

func TestQueryString(t *testing.T) {
	type Recv struct {
		X **struct {
			A []string  `api:"{query:'a'}"`
			B string    `api:"{query:'b'}"`
			C *[]string `api:"{query:'c'}{required:true}"`
			D *string   `api:"{query:'d'}"`
		}
		Y string  `api:"{query:'y'}{required:true}"`
		Z *string `api:"{query:'z'}"`
	}
	req := newRequest("http://localhost:8080/?a=a1&a=a2&b=b1&c=c1&c=c2&d=d1&d=d2&y=y1", nil, nil, nil)
	recv := new(Recv)
	binder := binding.New("api")
	err := binder.BindAndValidate(recv, req, nil)
	assert.Nil(t, err)
	assert.Equal(t, []string{"a1", "a2"}, (**recv.X).A)
	assert.Equal(t, "b1", (**recv.X).B)
	assert.Equal(t, []string{"c1", "c2"}, *(**recv.X).C)
	assert.Equal(t, "d1", *(**recv.X).D)
	assert.Equal(t, "y1", recv.Y)
	assert.Equal(t, (*string)(nil), recv.Z)
}

func TestQueryNum(t *testing.T) {
	type Recv struct {
		X **struct {
			A []int     `api:"{query:'a'}"`
			B int32     `api:"{query:'b'}"`
			C *[]uint16 `api:"{query:'c'}{required:true}"`
			D *float32  `api:"{query:'d'}"`
		}
		Y bool   `api:"{query:'y'}{required:true}"`
		Z *int64 `api:"{query:'z'}"`
	}
	req := newRequest("http://localhost:8080/?a=11&a=12&b=21&c=31&c=32&d=41&d=42&y=true", nil, nil, nil)
	recv := new(Recv)
	binder := binding.New("api")
	err := binder.BindAndValidate(recv, req, nil)
	assert.Nil(t, err)
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
			A []string  `api:"{header:'X-A'}"`
			B string    `api:"{header:'X-B'}"`
			C *[]string `api:"{header:'X-C'}{required:true}"`
			D *string   `api:"{header:'X-D'}"`
		}
		Y string  `api:"{header:'X-Y'}{required:true}"`
		Z *string `api:"{header:'X-Z'}"`
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
	binder := binding.New("api")
	err := binder.BindAndValidate(recv, req, nil)
	assert.Nil(t, err)
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
			A []int     `api:"{header:'X-A'}"`
			B int32     `api:"{header:'X-B'}"`
			C *[]uint16 `api:"{header:'X-C'}{required:true}"`
			D *float32  `api:"{header:'X-D'}"`
		}
		Y bool   `api:"{header:'X-Y'}{required:true}"`
		Z *int64 `api:"{header:'X-Z'}"`
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
	binder := binding.New("api")
	err := binder.BindAndValidate(recv, req, nil)
	assert.Nil(t, err)
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
			A []string  `api:"{cookie:'a'}"`
			B string    `api:"{cookie:'b'}"`
			C *[]string `api:"{cookie:'c'}{required:true}"`
			D *string   `api:"{cookie:'d'}"`
		}
		Y string  `api:"{cookie:'y'}{required:true}"`
		Z *string `api:"{cookie:'z'}"`
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
	binder := binding.New("api")
	err := binder.BindAndValidate(recv, req, nil)
	assert.Nil(t, err)
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
			A []int     `api:"{cookie:'a'}"`
			B int32     `api:"{cookie:'b'}"`
			C *[]uint16 `api:"{cookie:'c'}{required:true}"`
			D *float32  `api:"{cookie:'d'}"`
		}
		Y bool   `api:"{cookie:'y'}{required:true}"`
		Z *int64 `api:"{cookie:'z'}"`
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
	binder := binding.New("api")
	err := binder.BindAndValidate(recv, req, nil)
	assert.Nil(t, err)
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
			A []string  `api:"{body:'a'}"`
			B string    `api:"{body:'b'}"`
			C *[]string `api:"{body:'c'}{required:true}"`
			D *string   `api:"{body:'d'}"`
		}
		Y string  `api:"{body:'y'}{required:true}"`
		Z *string `api:"{body:'z'}"`
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
		binder := binding.New("api")
		err := binder.BindAndValidate(recv, req, nil)
		assert.Nil(t, err)
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
			A []int     `api:"{body:'a'}"`
			B int32     `api:"{body:'b'}"`
			C *[]uint16 `api:"{body:'c'}{required:true}"`
			D *float32  `api:"{body:'d'}"`
		}
		Y bool   `api:"{body:'y'}{required:true}"`
		Z *int64 `api:"{body:'z'}"`
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
		binder := binding.New("api")
		err := binder.BindAndValidate(recv, req, nil)
		assert.Nil(t, err)
		assert.Equal(t, []int{11, 12}, (**recv.X).A)
		assert.Equal(t, int32(-21), (**recv.X).B)
		assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
		assert.Equal(t, float32(41), *(**recv.X).D)
		assert.Equal(t, true, recv.Y)
		assert.Equal(t, (*int64)(nil), recv.Z)
	}
}

func TestJSON(t *testing.T) {
	type Recv struct {
		X **struct {
			A []string  `api:"{body:'a'}"`
			B int32     `api:""`
			C *[]uint16 `api:"{required:true}"`
			D *float32  `api:"{body:'d'}"`
		}
		Y string `api:"{body:'y'}{required:true}"`
		Z *int64 `api:""`
	}

	bodyReader := strings.NewReader(`{
		"X": {
			"a": ["a1","a2"],
			"B": 21,
			"C": [31,32],
			"d": 41
		},
		"y": "y1"
	}`)

	header := make(http.Header)
	header.Set("Content-Type", "application/json")
	req := newRequest("", header, nil, bodyReader)
	recv := new(Recv)
	binder := binding.New("api")
	err := binder.BindAndValidate(recv, req, nil)
	assert.Nil(t, err)
	assert.Equal(t, []string{"a1", "a2"}, (**recv.X).A)
	assert.Equal(t, int32(21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
	assert.Equal(t, float32(41), *(**recv.X).D)
	assert.Equal(t, "y1", recv.Y)
	assert.Equal(t, (*int64)(nil), recv.Z)
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
			A []string  `api:"{path:'a'}"`
			B int32     `api:"{path:'b'}"`
			C *[]uint16 `api:"{path:'c'}{required:true}"`
			D *float32  `api:"{path:'d'}"`
		}
		Y string `api:"{path:'y'}{required:true}"`
		Z *int64
	}

	req := newRequest("", nil, nil, nil)
	recv := new(Recv)
	binder := binding.New("api")
	err := binder.BindAndValidate(recv, req, new(testPathParams))
	assert.Nil(t, err)
	assert.Equal(t, []string{"a1"}, (**recv.X).A)
	assert.Equal(t, int32(-21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31}, (**recv.X).C)
	assert.Equal(t, float32(41), *(**recv.X).D)
	assert.Equal(t, "y1", recv.Y)
	assert.Equal(t, (*int64)(nil), recv.Z)
}

func TestAuto(t *testing.T) {
	type Recv struct {
		X **struct {
			A []string  `api:""`
			B int32     `api:""`
			C *[]uint16 `api:"{required:true}"`
			D *float32
		}
		Y string `api:"{required:true}"`
		Z *int64
	}
	query := make(url.Values)
	query.Add("A", "a1")
	query.Add("A", "a2")
	query.Add("B", "21")
	query.Add("Y", "y1")

	form := make(url.Values)
	form.Add("C", "31")
	form.Add("C", "32")
	form.Add("D", "41")
	form.Add("D", "42")
	for _, f := range []httpbody.Files{nil, {
		"f1": []httpbody.File{
			httpbody.NewFile("txt", strings.NewReader("f11 text.")),
		},
	}} {
		contentType, bodyReader := httpbody.NewFormBody2(form, f)
		header := make(http.Header)
		header.Set("Content-Type", contentType)
		req := newRequest("http://localhost/?"+query.Encode(), header, nil, bodyReader)
		recv := new(Recv)
		binder := binding.New("api").SetLevel(binding.Any)
		err := binder.BindAndValidate(recv, req, nil)
		assert.Nil(t, err)
		assert.Equal(t, []string{"a1", "a2"}, (**recv.X).A)
		assert.Equal(t, int32(21), (**recv.X).B)
		assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
		assert.Equal(t, float32(41), *(**recv.X).D)
		assert.Equal(t, "y1", recv.Y)
		assert.Equal(t, (*int64)(nil), recv.Z)
	}
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
