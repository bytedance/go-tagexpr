package binding

import (
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"github.com/bytedance/go-tagexpr/v2"
)

type in uint8

const (
	undefined in = iota
	raw_body
	path
	form
	query
	cookie
	header
	protobuf
	json
	default_val
	maxIn
)

var (
	allIn = func() []in {
		a := []in{}
		for i := undefined + 1; i < maxIn; i++ {
			a = append(a, i)
		}
		return a
	}()
	sortedDefaultIn = func() []in {
		var a []in
		for i := path; i <= json; i++ {
			a = append(a, i)
		}
		return a
	}()
)

type codec in

const (
	bodyUnsupport = codec(0)
	bodyForm      = codec(form)
	bodyJSON      = codec(json)
	bodyProtobuf  = codec(protobuf)
)

type receiver struct {
	hasPath, hasQuery, hasBody, hasCookie, hasDefaultVal, hasVd bool

	params []*paramInfo

	looseZeroMode bool
}

func (r *receiver) assginIn(i in, v bool) {
	switch i {
	case path:
		r.hasPath = r.hasPath || v
	case query:
		r.hasQuery = r.hasQuery || v
	case form, json, protobuf, raw_body:
		r.hasBody = r.hasBody || v
	case cookie:
		r.hasCookie = r.hasCookie || v
	case default_val:
		r.hasDefaultVal = r.hasDefaultVal || v
	}
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
	p = &paramInfo{
		fieldSelector:  fieldSelector,
		structField:    fh.StructField(),
		omitIns:        make(map[in]bool, maxIn),
		bindErrFactory: bindErrFactory,
		looseZeroMode:  r.looseZeroMode,
	}
	r.params = append(r.params, p)
	return p
}

func (r *receiver) getBodyInfo(req Request) (codec, []byte, error) {
	if r.hasBody {
		c := getBodyCodec(req)
		b, err := req.GetBody()
		return c, b, err
	}
	return bodyUnsupport, nil, nil
}

func (r *receiver) prebindBody(pointer interface{}, val reflect.Value, bodyCodec codec, bodyBytes []byte) error {
	switch bodyCodec {
	case bodyJSON:
		return bindJSON(pointer, bodyBytes)
	case bodyProtobuf:
		return bindProtobuf(pointer, bodyBytes)
	default:
		return nil
	}
}

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

func (r *receiver) getQuery(req Request) url.Values {
	if r.hasQuery {
		return req.GetQuery()
	}
	return nil
}

func (r *receiver) getCookies(req Request) []*http.Cookie {
	if r.hasCookie {
		return req.GetCookies()
	}
	return nil
}

var replacer = strings.NewReplacer("\x01", "[]", "\x02", "{}")

func (r *receiver) initParams() error {
	names := make(map[string][maxIn]string, len(r.params))

	for _, p := range r.params {
		if p.structField.Anonymous {
			continue
		}
		a := [maxIn]string{}
		for _, paramIn := range allIn {
			paramName := p.name(paramIn)
			// check paramName
			if strings.ContainsAny(paramName, ".\x01\x02") {
				return fmt.Errorf("field \"%s\" has illegal paramName \"%s\" which contains reserved characters", p.fieldSelector, paramName)
			}
			a[paramIn] = paramName
		}
		names[p.fieldSelector] = a
	}

	for _, p := range r.params {
		paths, _ := tagexpr.FieldSelector(p.fieldSelector).Split()
		for _, info := range p.tagInfos {
			var fs string
			for _, s := range paths {

				if fs == "" {
					fs = s
				} else {
					fs = tagexpr.JoinFieldSelector(fs, s)
				}
				var name string
				// parent elemKind is indirect
				if strings.LastIndexAny(s, "\x01\x02") == len(s)-1 {
					name = names[fs[:len(fs)-1]][info.paramIn] + s[len(s)-1:]
				} else {
					name = names[fs][info.paramIn]
				}
				if name != "" {
					info.namePath += name + "."
				}
			}
			info.namePath = info.namePath + p.name(info.paramIn)
			infoShowPath := replacer.Replace(info.namePath)
			info.requiredError = p.bindErrFactory(infoShowPath, "missing required parameter")
			info.typeError = p.bindErrFactory(infoShowPath, "parameter type does not match binding data")
			info.cannotError = p.bindErrFactory(infoShowPath, "parameter cannot be bound")
			info.contentTypeError = p.bindErrFactory(infoShowPath, "does not support binding to the content type body")
		}
		p.setDefaultVal()
	}
	return nil
}
