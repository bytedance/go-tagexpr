package binding

import (
	"net/http"
	"net/url"
	"strings"
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

func (r *receiver) getOrAddParam(fieldSelector string) *paramInfo {
	for _, p := range r.params {
		if p.fieldSelector == fieldSelector {
			return p
		}
	}
	p := new(paramInfo)
	p.fieldSelector = fieldSelector
	r.params = append(r.params, p)
	return p
}

func (r *receiver) getBodyCodec(req *http.Request) uint8 {
	ct := req.Header.Get("Content-Type")
	switch ct {
	case "application/json":
		return jsonBody
	case "application/x-www-form-urlencoded":
		return formBody
	default:
		if strings.HasPrefix(ct, "multipart/form-data") {
			return formBody
		}
		return unsupportBody
	}
}

func (r *receiver) getBodyBytes(req *http.Request, must bool) ([]byte, error) {
	if must || r.hasRawBody {
		return copyBody(req)
	}
	return nil, nil
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

// func (a *receiver) getPath(req *http.Request) *url.Values {
// 	v := new(url.Values)
// 	if a.hasQuery {
// 		(*v) = req.URL.Query()
// 	}
// 	return v
// }
