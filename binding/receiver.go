package binding

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/bytedance/go-tagexpr"
	"github.com/henrylee2cn/goutil"
)

const (
	auto uint8 = iota
	query
	path
	header
	cookie
	body
	raw_body
)

const (
	unsupportBody uint8 = iota
	jsonBody
	formBody
)

type receiver struct {
	hasAuto, hasQuery, hasCookie, hasPath, hasBody, hasRawBody, hasVd bool

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
	p.fieldSelector = fieldSelector
	p.structField = fh.StructField()
	p.bindErrFactory = bindErrFactory
	r.params = append(r.params, p)
	return p
}

func (r *receiver) getBodyCodec(req *http.Request) uint8 {
	ct := req.Header.Get("Content-Type")
	idx := strings.Index(ct, ";")
	if idx != -1 {
		ct = strings.TrimRight(ct[:idx], " ")
	}
	switch ct {
	case "application/json":
		return jsonBody
	case "application/x-www-form-urlencoded", "multipart/form-data":
		return formBody
	default:
		return unsupportBody
	}
}

func (r *receiver) getBody(req *http.Request, must bool) ([]byte, string, error) {
	if must || r.hasRawBody {
		bodyBytes, err := copyBody(req)
		if err == nil {
			return bodyBytes, goutil.BytesToString(bodyBytes), nil
		}
		return bodyBytes, "", nil
	}
	return nil, "", nil
}

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

func (r *receiver) getPostForm(req *http.Request, must bool) (url.Values, error) {
	if must {
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
