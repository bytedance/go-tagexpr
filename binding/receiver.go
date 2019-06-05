package binding

import (
	"net/http"
	"net/url"
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

type receiver struct {
	hasAuto, hasQuery, hasPath, hasBody, hasRawBody, hasVd bool

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

func (r *receiver) getBodyBytes(req *http.Request) ([]byte, error) {
	if r.hasRawBody {
		return copyBody(req)
	}
	return nil, nil
}

func (r *receiver) getQuery(req *http.Request) url.Values {
	if r.hasQuery {
		return req.URL.Query()
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
