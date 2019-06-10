package binding

import (
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/bytedance/go-tagexpr"
	"github.com/bytedance/go-tagexpr/binding/jsonparam"
	"github.com/gogo/protobuf/proto"
	"github.com/henrylee2cn/goutil"
	"github.com/tidwall/gjson"
)

const (
	auto uint8 = iota
	query
	path
	header
	cookie
	rawBody
	form
	otherBody
)

const (
	bodyUnsupport int8 = iota
	bodyForm
	bodyJSON
	bodyProtobuf
)

type receiver struct {
	hasAuto, hasQuery, hasCookie, hasPath, hasForm, hasBody, hasVd bool

	params []*paramInfo
}

func (r *receiver) getParam(fieldSelector string) *paramInfo {
	for _, p := range r.params {
		if p.fieldSelector == fieldSelector {
			return p
		}
	}
	return nil
}

func (r *receiver) getOrAddParam(fh *tagexpr.FieldHandler, bindErrFactory func(failField, msg string) error) *paramInfo {
	fieldSelector := fh.StringSelector()
	p := r.getParam(fieldSelector)
	if p != nil {
		return p
	}
	p = new(paramInfo)
	p.in = auto
	p.fieldSelector = fieldSelector
	p.structField = fh.StructField()
	p.name = p.structField.Name
	p.bindErrFactory = bindErrFactory
	r.params = append(r.params, p)
	return p
}

func (r *receiver) getBodyCodec(req *http.Request) int8 {
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

func (r *receiver) getBody(req *http.Request) ([]byte, string, error) {
	if r.hasBody {
		bodyBytes, err := copyBody(req)
		if err == nil {
			return bodyBytes, goutil.BytesToString(bodyBytes), nil
		}
		return bodyBytes, "", nil
	}
	return nil, "", nil
}

func (r *receiver) bindOtherBody(structPointer interface{}, value reflect.Value, bodyCodec int8, bodyBytes []byte) error {
	switch bodyCodec {
	case bodyJSON:
		jsonparam.Assign(gjson.Parse(goutil.BytesToString(bodyBytes)), value)
	case bodyProtobuf:
		msg, ok := structPointer.(proto.Message)
		if !ok {
			return errors.New("protobuf content type is not supported")
		}
		if err := proto.Unmarshal(bodyBytes, msg); err != nil {
			return err
		}
	}
	return nil
}

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

func (r *receiver) getPostForm(req *http.Request, bodyCodec int8) (url.Values, error) {
	if bodyCodec == bodyForm && (r.hasForm || r.hasBody) {
		if req.PostForm == nil {
			req.ParseMultipartForm(defaultMaxMemory)
		}
		return req.Form, nil
	}
	return nil, nil
}

func (r *receiver) getQuery(req *http.Request) url.Values {
	if r.hasQuery {
		return req.URL.Query()
	}
	return nil
}

func (r *receiver) getCookies(req *http.Request) []*http.Cookie {
	if r.hasCookie {
		return req.Cookies()
	}
	return nil
}

func (r *receiver) initParams() {
	if !r.hasBody {
		return
	}
	names := make(map[string]string, len(r.params))
	for _, p := range r.params {
		if !p.structField.Anonymous {
			names[p.fieldSelector] = p.name
		}
	}
	for _, p := range r.params {
		paths, _ := tagexpr.FieldSelector(p.fieldSelector).Split()
		var fs, namePath string
		for _, s := range paths {
			if fs == "" {
				fs = s
			} else {
				fs = tagexpr.JoinFieldSelector(fs, s)
			}
			name := names[fs]
			if name != "" {
				namePath = name + "."
			}
		}
		p.namePath = namePath + p.name
		p.requiredError = p.bindErrFactory(p.namePath, "missing required parameter")
		p.typeError = p.bindErrFactory(p.namePath, "parameter type does not match binding data")
		p.cannotError = p.bindErrFactory(p.namePath, "parameter cannot be bound")
		p.contentTypeError = p.bindErrFactory(p.namePath, "does not support binding to the content type body")
	}
}
