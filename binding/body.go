package binding

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	jsonpkg "encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"google.golang.org/protobuf/proto"
)

func getBodyCodec(req Request) codec {
	ct := req.GetContentType()
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

// Bytes returns all of the body bytes.
func (b *Body) Bytes() []byte {
	return b.bodyBytes
}

// Len returns all of the body length.
func (b *Body) Len() int {
	return len(b.bodyBytes)
}

// GetBody get the body from http.Request
func GetBody(r *http.Request) (*Body, error) {
	switch body := r.Body.(type) {
	case *Body:
		body.Reset()
		return body, nil
	default:
		var err error
		var closeBody = r.Body.Close
		switch r.Header.Get("Content-Encoding") {
		case "gzip":
			var gzipReader *gzip.Reader
			gzipReader, err = gzip.NewReader(r.Body)
			if err == nil {
				r.Body = gzipReader
			}
		case "deflate":
			r.Body = flate.NewReader(r.Body)
		case "zlib":
			var readCloser io.ReadCloser
			readCloser, err = zlib.NewReader(r.Body)
			if err == nil {
				r.Body = readCloser
			}
		}
		if err != nil {
			return nil, err
		}
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r.Body)
		_ = closeBody()
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
