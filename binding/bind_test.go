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
			F **string `api:"{raw_body:nil}"`
		}
		S string `api:"{raw_body:nil}"`
	}
	bodyBytes := []byte("rawbody.............")
	req := newRequest("", "", nil, bodyBytes)
	recv := new(Recv)
	binder := New("api")
	err := binder.BindAndValidate(req, recv)
	assert.Nil(t, err)
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
	req := newRequest("http://localhost:8080/?a=a1&a=a2&b=b1&c=c1&c=c2&d=d1&d=d2&y=y1", "", nil, nil)
	recv := new(Recv)
	binder := New("api")
	err := binder.BindAndValidate(req, recv)
	assert.Nil(t, err)
	assert.Equal(t, []string{"a1", "a2"}, (**recv.X).A)
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
	req := newRequest("http://localhost:8080/?a=11&a=12&b=21&c=31&c=32&d=41&d=42&y=51", "", nil, nil)
	recv := new(Recv)
	binder := New("api")
	err := binder.BindAndValidate(req, recv)
	assert.Nil(t, err)
	assert.Equal(t, []int{11, 12}, (**recv.X).A)
	assert.Equal(t, int32(21), (**recv.X).B)
	assert.Equal(t, &[]uint16{31, 32}, (**recv.X).C)
}

func newRequest(u string, contentType string, cookies []*http.Cookie, body []byte) *http.Request {
	urlObj, _ := url.Parse(u)
	req := &http.Request{
		URL:    urlObj,
		Body:   ioutil.NopCloser(bytes.NewReader(body)),
		Header: make(http.Header),
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}
