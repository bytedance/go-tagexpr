package binding

import (
	"bytes"
	"errors"
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
			req.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))
		}
		return bodyBytes, err
	default:
		return nil, nil
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
	req.Body = ioutil.NopCloser(bytes.NewReader(b))
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
