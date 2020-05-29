package binding

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gogo/protobuf/proto"
	jsonpkg "github.com/json-iterator/go"
)

func getBodyInfo(req *http.Request) (codec, []byte, error) {
	bodyCodec := getBodyCodec(req)
	bodyBytes, err := getBody(req, bodyCodec)
	return bodyCodec, bodyBytes, err
}

func getBodyCodec(req *http.Request) codec {
	ct := req.Header.Get("Content-Type")
	idx := strings.Index(ct, ";")
	if idx != -1 {
		ct = strings.TrimRight(ct[:idx], " ")
	}
	switch ct {
	case "application/json":
		return bodyJSON
	case "application/x-protobuf":
		return bodyProtobuf
	case "application/x-www-form-urlencoded", "multipart/form-data":
		return bodyForm
	default:
		return bodyUnsupport
	}
}

func getBody(req *http.Request, bodyCodec codec) ([]byte, error) {
	switch req.Method {
	case "POST", "PUT", "PATCH", "DELETE":
		body, err := copyBody(req)
		if err == nil && bodyCodec == bodyForm && req.PostForm == nil {
			req.ParseMultipartForm(defaultMaxMemory)
			body.renew()
		}
		return body.bodyBytes, err
	default:
		return nil, nil
	}
}

type Body struct {
	*bytes.Buffer
	bodyBytes []byte
}

func (Body) Close() error { return nil }

func (b *Body) renew() {
	b.Buffer.Reset()
	b.Buffer = bytes.NewBuffer(b.bodyBytes)
}

// GetCopiedBody after binding, try to quickly extract the body from http.Request
func GetCopiedBody(r *http.Request) ([]byte, bool) {
	body, ok := r.Body.(*Body)
	if ok {
		return body.bodyBytes, true
	}
	return nil, r.Body == nil
}

func newBody(body *bytes.Buffer) *Body {
	return &Body{
		Buffer:    body,
		bodyBytes: body.Bytes(),
	}
}

func copyBody(req *http.Request) (*Body, error) {
	if req.Body == nil {
		return nil, nil
	}
	var buf bytes.Buffer
	_, err := io.Copy(&buf, req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	body := newBody(&buf)
	req.Body = body
	return body, nil
}

func bindJSON(pointer interface{}, bodyBytes []byte) error {
	if jsonUnmarshalFunc != nil {
		return jsonUnmarshalFunc(bodyBytes, pointer)
	}
	return jsonpkg.Unmarshal(bodyBytes, pointer)
}

func bindProtobuf(pointer interface{}, bodyBytes []byte) error {
	msg, ok := pointer.(proto.Message)
	if !ok {
		return errors.New("protobuf content type is not supported")
	}
	return proto.Unmarshal(bodyBytes, msg)
}
