package binding

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
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
		bodyBytes, err := copyBody(req)
		if err == nil && bodyCodec == bodyForm && req.PostForm == nil {
			req.ParseMultipartForm(defaultMaxMemory)
			req.Body = newBody(bodyBytes)
		}
		return bodyBytes, err
	default:
		return nil, nil
	}
}

type Body struct {
	io.Reader
	bodyBytes []byte
}

func (Body) Close() error { return nil }

// GetCopiedBody after binding, try to quickly extract the body from http.Request
func GetCopiedBody(r *http.Request) ([]byte, bool) {
	body, ok := r.Body.(*Body)
	if ok {
		return body.bodyBytes, true
	}
	return nil, r.Body == nil
}

func newBody(bodyBytes []byte) io.ReadCloser {
	return &Body{
		Reader:    bytes.NewReader(bodyBytes),
		bodyBytes: bodyBytes,
	}
}

func copyBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}
	b, err := ioutil.ReadAll(req.Body)
	req.Body.Close()
	if err != nil {
		return nil, err
	}
	req.Body = newBody(b)
	return b, nil
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
