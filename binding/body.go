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
		body, err := GetBody(req)
		if err == nil && bodyCodec == bodyForm && req.PostForm == nil {
			req.ParseMultipartForm(defaultMaxMemory)
			body.Reset()
		}
		return body.bodyBytes, err
	default:
		return nil, nil
	}
}

// Body body copy
type Body struct {
	*bytes.Buffer
	bodyBytes []byte
}

// Close close.
func (Body) Close() error { return nil }

// Reset zero offset.
func (b *Body) Reset() {
	b.Buffer = bytes.NewBuffer(b.bodyBytes)
}

// Bytes returns all of body bytes.
func (b *Body) Bytes() []byte {
	return b.bodyBytes
}

// GetBody get the body from http.Request
func GetBody(r *http.Request) (*Body, error) {
	switch body := r.Body.(type) {
	case *Body:
		body.Reset()
		return body, nil
	default:
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r.Body)
		r.Body.Close()
		if err != nil {
			return nil, err
		}
		_body := &Body{
			Buffer:    &buf,
			bodyBytes: buf.Bytes(),
		}
		r.Body = _body
		return _body, nil
	}
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
