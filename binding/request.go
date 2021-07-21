package binding

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
)

type requestWithFileHeader interface {
	Request
	GetFileHeaders() (map[string][]*multipart.FileHeader, error)
}

type Request interface {
	GetMethod() string
	GetQuery() url.Values
	GetContentType() string
	GetHeader() http.Header
	GetCookies() []*http.Cookie
	GetBody() ([]byte, error)
	GetPostForm() (url.Values, error)
	GetForm() (url.Values, error)
}

func wrapRequest(req *http.Request) Request {
	r := &httpRequest{
		Request: req,
	}
	if getBodyCodec(r) == bodyForm && req.PostForm == nil {
		b, _ := r.GetBody()
		if b != nil {
			req.ParseMultipartForm(defaultMaxMemory)
		}
	}
	return r
}

type httpRequest struct {
	*http.Request
}

func (r *httpRequest) GetMethod() string {
	return r.Method
}
func (r *httpRequest) GetQuery() url.Values {
	return r.URL.Query()
}

func (r *httpRequest) GetContentType() string {
	return r.GetHeader().Get("Content-Type")
}

func (r *httpRequest) GetHeader() http.Header {
	return r.Header
}

func (r *httpRequest) GetCookies() []*http.Cookie {
	return r.Cookies()
}

func (r *httpRequest) GetBody() ([]byte, error) {
	body, _ := r.Body.(*Body)
	if body != nil {
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
		body = &Body{
			Buffer:    &buf,
			bodyBytes: buf.Bytes(),
		}
		r.Body = body
		return body.bodyBytes, nil
	default:
		return nil, nil
	}
}

func (r *httpRequest) GetPostForm() (url.Values, error) {
	return r.PostForm, nil
}

func (r *httpRequest) GetForm() (url.Values, error) {
	return r.Form, nil
}

func (r *httpRequest) GetFileHeaders() (map[string][]*multipart.FileHeader, error) {
	if r.MultipartForm == nil {
		return nil, nil
	}
	return r.MultipartForm.File, nil
}
