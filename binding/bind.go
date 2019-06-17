package binding

import (
	"net/http"
	"reflect"
	"sync"

	"github.com/bytedance/go-tagexpr"
	"github.com/bytedance/go-tagexpr/validator"
	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/tpack"
)

// Binding binding and verification tool for http request
type Binding struct {
	vd             *validator.Validator
	recvs          map[int32]*receiver
	lock           sync.RWMutex
	bindErrFactory func(failField, msg string) error
	tagNames       TagNames
}

// New creates a binding tool.
// NOTE:
//  Use default tag name for tagNames fields that are empty
func New(tagNames *TagNames) *Binding {
	if tagNames == nil {
		tagNames = new(TagNames)
	}
	b := &Binding{
		recvs:    make(map[int32]*receiver, 1024),
		tagNames: *tagNames,
	}
	b.tagNames.init()
	b.vd = validator.New(b.tagNames.Validator)
	return b.SetErrorFactory(nil, nil)
}

var defaultValidatingErrFactory = newDefaultErrorFactory("validating")
var defaultBindErrFactory = newDefaultErrorFactory("binding")

// SetErrorFactory customizes the factory of validation error.
// NOTE:
//  If errFactory==nil, the default is used
func (b *Binding) SetErrorFactory(bindErrFactory, validatingErrFactory func(failField, msg string) error) *Binding {
	if bindErrFactory == nil {
		bindErrFactory = defaultBindErrFactory
	}
	if validatingErrFactory == nil {
		validatingErrFactory = defaultValidatingErrFactory
	}
	b.bindErrFactory = bindErrFactory
	b.vd.SetErrorFactory(validatingErrFactory)
	return b
}

// BindAndValidate binds the request parameters and validates them if needed.
func (b *Binding) BindAndValidate(structPointer interface{}, req *http.Request, pathParams PathParams) error {
	v, hasVd, err := b.bind(structPointer, req, pathParams)
	if err != nil {
		return err
	}
	if hasVd {
		return b.vd.Validate(v)
	}
	return nil
}

// Bind binds the request parameters.
func (b *Binding) Bind(structPointer interface{}, req *http.Request, pathParams PathParams) error {
	_, _, err := b.bind(structPointer, req, pathParams)
	return err
}

// Validate validates whether the fields of value is valid.
func (b *Binding) Validate(value interface{}) error {
	return b.vd.Validate(value)
}

func (b *Binding) bind(structPointer interface{}, req *http.Request, pathParams PathParams) (value reflect.Value, hasVd bool, err error) {
	value, err = b.structValueOf(structPointer)
	if err != nil {
		return
	}
	recv, err := b.getOrPrepareReceiver(value)
	if err != nil {
		return
	}

	expr, err := b.vd.VM().Run(value)
	if err != nil {
		return
	}

	bodyCodec := recv.getBodyCodec(req)

	bodyBytes, bodyString, err := recv.getBody(req)
	if err != nil {
		return
	}
	err = recv.prebindBody(structPointer, value, bodyCodec, bodyBytes)
	if err != nil {
		return
	}

	postForm, err := recv.getPostForm(req, bodyCodec)
	if err != nil {
		return
	}

	queryValues := recv.getQuery(req)
	cookies := recv.getCookies(req)

	for _, param := range recv.params {
		for _, info := range param.tagInfos {
			switch info.paramIn {
			case query:
				_, err = param.bindQuery(info, expr, queryValues)
			case path:
				_, err = param.bindPath(info, expr, pathParams)
			case header:
				_, err = param.bindHeader(info, expr, req.Header)
			case cookie:
				err = param.bindCookie(info, expr, cookies)
			case rawBody:
				err = param.bindRawBody(info, expr, bodyBytes)
			case form, json, protobuf:
				if info.paramIn == in(bodyCodec) {
					_, err = param.bindOrRequireBody(info, expr, bodyCodec, bodyString, postForm)
				}
			case auto:
				// Try bind parameters from the body when the request has body,
				// otherwise try bind from the URL query
				if len(bodyBytes) == 0 {
					if !param.omitIns[query] {
						if queryValues == nil {
							queryValues = req.URL.Query()
						}
						_, err = param.bindQuery(info, expr, queryValues)
					}
				} else {
					switch bodyCodec {
					case bodyForm:
						if !param.omitIns[form] {
							_, err = param.bindMapStrings(info, expr, postForm)
						}
					case bodyJSON:
						if !param.omitIns[json] {
							err = param.checkRequireJSON(info, expr, bodyString, false)
						}
					case bodyProtobuf:
						if !param.omitIns[protobuf] {
							err = param.checkRequireProtobuf(info, expr, false)
						}
					}
				}
			}
			if err != nil {
				return value, recv.hasVd, err
			}
		}
	}
	return value, recv.hasVd, nil
}

func (b *Binding) structValueOf(structPointer interface{}) (reflect.Value, error) {
	v, ok := structPointer.(reflect.Value)
	if !ok {
		v = reflect.ValueOf(structPointer)
	}
	if v.Kind() != reflect.Ptr {
		return v, b.bindErrFactory("", "structPointer must be a non-nil struct pointer")
	}
	v = goutil.DereferenceValue(v)
	if v.Kind() != reflect.Struct || !v.CanAddr() || !v.IsValid() {
		return v, b.bindErrFactory("", "structPointer must be a non-nil struct pointer")
	}
	return v, nil
}

func (b *Binding) getOrPrepareReceiver(value reflect.Value) (*receiver, error) {
	runtimeTypeID := tpack.From(value).RuntimeTypeID()
	b.lock.RLock()
	recv, ok := b.recvs[runtimeTypeID]
	b.lock.RUnlock()
	if ok {
		return recv, nil
	}

	expr, err := b.vd.VM().Run(reflect.New(value.Type()).Elem())
	if err != nil {
		return nil, err
	}
	recv = &receiver{
		params: make([]*paramInfo, 0, 16),
	}
	var errExprSelector tagexpr.ExprSelector
	var errMsg string

	expr.RangeFields(func(fh *tagexpr.FieldHandler) bool {
		if !fh.Value(true).CanSet() {
			selector := fh.StringSelector()
			errMsg = "field cannot be set: " + selector
			errExprSelector = tagexpr.ExprSelector(selector)
			return false
		}

		tagKVs := b.tagNames.parse(fh.StructField())
		p := recv.getOrAddParam(fh, b.bindErrFactory)
		tagInfos := [maxIn]*tagInfo{}
	L:
		for _, tagKV := range tagKVs {
			paramIn := auto
			switch tagKV.name {
			case b.tagNames.Validator:
				recv.hasVd = true
				continue L

			case b.tagNames.Query:
				recv.hasQuery = true
				paramIn = query
			case b.tagNames.PathParam:
				recv.hasPath = true
				paramIn = path
			case b.tagNames.Header:
				paramIn = header
			case b.tagNames.Cookie:
				recv.hasCookie = true
				paramIn = cookie
			case b.tagNames.RawBody:
				recv.hasBody = true
				paramIn = rawBody
			case b.tagNames.FormBody:
				recv.hasBody = true
				paramIn = form
			case b.tagNames.protobufBody:
				recv.hasBody = true
				paramIn = protobuf
			case b.tagNames.jsonBody:
				recv.hasBody = true
				paramIn = json

			default:
				continue L
			}
			tagInfos[paramIn] = tagKV.defaultSplit()
		}
		for i, info := range tagInfos {
			if info != nil {
				if info.paramName == "-" {
					p.omitIns[in(i)] = true
				} else {
					info.paramIn = in(i)
					p.tagInfos = append(p.tagInfos, info)
				}
			}
		}
		if len(p.tagInfos) == 0 {
			p.tagInfos = append(p.tagInfos, &tagInfo{
				paramIn:   auto,
				paramName: p.structField.Name,
			})
			recv.hasBody = true
		}
		if !recv.hasVd {
			_, recv.hasVd = tagKVs.lookup(b.tagNames.Validator)
		}
		return true
	})

	if errMsg != "" {
		return nil, b.bindErrFactory(errExprSelector.String(), errMsg)
	}

	recv.initParams()

	b.lock.Lock()
	b.recvs[runtimeTypeID] = recv
	b.lock.Unlock()

	return recv, nil
}
