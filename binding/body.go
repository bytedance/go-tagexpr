package binding

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"errors"
	"io"
	"net/http"
	"strings"

	"google.golang.org/protobuf/proto"
)

func getBodyCodec(req Request) codec {
	// according to rfc7231 https://datatracker.ietf.org/doc/html/rfc7231#section-3.1.1.5
	// content type just for payload and payload for Http GET Method are meanless; this 
	// will cause bad case for http GET method with a muanual added Content-Type Header
	if req.GetMethod() == http.MethodGet {
		return bodyUnsupport
	} 
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
func (b *Body) Close() error {
	b.Buffer = nil
	b.bodyBytes = nil
	return nil
}

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
		// Maximum size of decompressed body size. Currently 256 MB
		var maximumSizeLimit int64 = 256 * 1024 * 1024
		_, _ = io.CopyN(&buf, r.Body, maximumSizeLimit)
		_ = closeBody()
		_body := &Body{
			Buffer:    &buf,
			bodyBytes: buf.Bytes(),
		}
		r.Body = _body
		return _body, nil
	}
}

func bindProtobuf(pointer interface{}, bodyBytes []byte) error {
	msg, ok := pointer.(proto.Message)
	if !ok {
		return errors.New("protobuf content type is not supported")
	}
	return proto.Unmarshal(bodyBytes, msg)
}
