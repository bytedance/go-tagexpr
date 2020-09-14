package binding

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
)

type Request interface {
	GetQuery() url.Values
	GetPostForm() (url.Values, error)
	GetForm() (url.Values, error)
	GetCookies() []*http.Cookie
	GetHeader() http.Header
	GetMethod() string
	GetBody() ([]byte, error)
}

func wrapRequest(req *http.Request) Request {
	r := &httpRequest{
		Request: req,
	}
	if getBodyCodec(r) == bodyForm && req.PostForm == nil {
		_, _ = r.GetBody()
		if r.body != nil {
			req.ParseMultipartForm(defaultMaxMemory)
		}
	}
	return r
}

type httpRequest struct {
	*http.Request
	body *Body
}

func (r *httpRequest) GetQuery() url.Values {
	return r.URL.Query()
}

func (r *httpRequest) GetPostForm() (url.Values, error) {
	return r.PostForm, nil
}

func (r *httpRequest) GetForm() (url.Values, error) {
	return r.Form, nil
}

func (r *httpRequest) GetCookies() []*http.Cookie {
	return r.Cookies()
}

func (r *httpRequest) GetHeader() http.Header {
	return r.Header
}

func (r *httpRequest) GetMethod() string {
	return r.Method
}

func (r *httpRequest) GetBody() ([]byte, error) {
	if r.body != nil {
		r.body.Reset()
		return r.body.bodyBytes, nil
	}
	body, ok := r.Body.(*Body)
	if ok {
		r.body = body
		body.Reset()
		return body.bodyBytes, nil
	}
	switch r.Method {
	case "POST", "PUT", "PATCH", "DELETE":
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r.Body)
		r.Body.Close()
		if err != nil {
			return nil, err
		}
		r.body = &Body{
			Buffer:    &buf,
			bodyBytes: buf.Bytes(),
		}
		r.Body = r.body
		return r.body.bodyBytes, nil
	default:
		return nil, nil
	}
}
