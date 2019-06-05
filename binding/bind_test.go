package binding

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

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
	req := newRequest("", nil, nil, bodyBytes)
	recv := new(Recv)
	binder := New("api")
	err := binder.BindAndValidate(req, recv)
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
	binder := New("api")
	err := binder.BindAndValidate(req, recv)
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
			D *uint     `api:"{query:'d'}"`
		}
		Y int8   `api:"{query:'y'}{required:true}"`
		Z *int64 `api:"{query:'z'}"`
	}
	req := newRequest("http://localhost:8080/?a=11&a=12&b=21&c=31&c=32&d=41&d=42&y=51", nil, nil, nil)
	recv := new(Recv)
	binder := New("api")
	err := binder.BindAndValidate(req, recv)
	assert.Nil(t, err)
	assert.Equal(t, []int{11, 12}, (**recv.X).A)
	assert.Equal(t, int32(21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
	assert.Equal(t, uint(41), *(**recv.X).D)
	assert.Equal(t, int8(51), recv.Y)
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
	binder := New("api")
	err := binder.BindAndValidate(req, recv)
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
			D *uint     `api:"{header:'X-D'}"`
		}
		Y int8   `api:"{header:'X-Y'}{required:true}"`
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
	header.Add("X-Y", "51")
	req := newRequest("", header, nil, nil)
	recv := new(Recv)
	binder := New("api")
	err := binder.BindAndValidate(req, recv)
	assert.Nil(t, err)
	assert.Equal(t, []int{11, 12}, (**recv.X).A)
	assert.Equal(t, int32(21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
	assert.Equal(t, uint(41), *(**recv.X).D)
	assert.Equal(t, int8(51), recv.Y)
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
	binder := New("api")
	err := binder.BindAndValidate(req, recv)
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
			D *uint     `api:"{cookie:'d'}"`
		}
		Y int8   `api:"{cookie:'y'}{required:true}"`
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
		{Name: "y", Value: "51"},
	}
	req := newRequest("", nil, cookies, nil)
	recv := new(Recv)
	binder := New("api")
	err := binder.BindAndValidate(req, recv)
	assert.Nil(t, err)
	assert.Equal(t, []int{11, 12}, (**recv.X).A)
	assert.Equal(t, int32(21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
	assert.Equal(t, uint(41), *(**recv.X).D)
	assert.Equal(t, int8(51), recv.Y)
	assert.Equal(t, (*int64)(nil), recv.Z)
}

func newRequest(u string, header http.Header, cookies []*http.Cookie, body []byte) *http.Request {
	if header == nil {
		header = make(http.Header)
	}
	urlObj, _ := url.Parse(u)
	req := &http.Request{
		URL:    urlObj,
		Body:   ioutil.NopCloser(bytes.NewReader(body)),
		Header: header,
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}
