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
	config         Config
}

// New creates a binding tool.
// NOTE:
//  Use default tag name for config fields that are empty
func New(config *Config) *Binding {
	if config == nil {
		config = new(Config)
	}
	b := &Binding{
		recvs:  make(map[int32]*receiver, 1024),
		config: *config,
	}
	b.config.init()
	b.vd = validator.New(b.config.Validator)
	return b.SetErrorFactory(nil, nil)
}

// SetLooseZeroMode if set to true,
// the empty string request parameter is bound to the zero value of parameter.
// NOTE:
//  The default is false;
//  Suitable for these parameter types: query/header/cookie/form .
func (b *Binding) SetLooseZeroMode(enable bool) *Binding {
	b.config.LooseZeroMode = enable
	for k := range b.recvs {
		delete(b.recvs, k)
	}
	return b
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

		for i, info := range param.tagInfos {
			var found bool
			switch info.paramIn {
			case path:
				found, err = param.bindPath(info, expr, pathParams)
			case query:
				found, err = param.bindQuery(info, expr, queryValues)
			case cookie:
				err = param.bindCookie(info, expr, cookies)
				found = err == nil
			case header:
				found, err = param.bindHeader(info, expr, req.Header)
			case form, json, protobuf:
				if info.paramIn == in(bodyCodec) {
					found, err = param.bindOrRequireBody(info, expr, bodyCodec, bodyString, postForm)
				} else if info.required {
					found = false
					err = info.requiredError
				}
			case raw_body:
				err = param.bindRawBody(info, expr, bodyBytes)
				found = err == nil
			}
			if found && err == nil {
				break
			}
			if (found || i == len(param.tagInfos)-1) && err != nil {
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
		params:        make([]*paramInfo, 0, 16),
		looseZeroMode: b.config.LooseZeroMode,
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

		tagKVs := b.config.parse(fh.StructField())
		p := recv.getOrAddParam(fh, b.bindErrFactory)
		tagInfos := [maxIn]*tagInfo{}
	L:
		for _, tagKV := range tagKVs {
			paramIn := undefined
			switch tagKV.name {
			case b.config.Validator:
				recv.hasVd = true
				continue L
			case b.config.PathParam:
				paramIn = path
			case b.config.FormBody:
				paramIn = form
			case b.config.Query:
				paramIn = query
			case b.config.Cookie:
				paramIn = cookie
			case b.config.Header:
				paramIn = header
			case b.config.protobufBody:
				paramIn = protobuf
			case b.config.jsonBody:
				paramIn = json
			case b.config.RawBody:
				paramIn = raw_body
			default:
				continue L
			}
			tagInfos[paramIn] = tagKV.defaultSplit()
		}
		for i, info := range tagInfos {
			if info != nil {
				if info.paramName == "-" {
					p.omitIns[in(i)] = true
					recv.assginIn(in(i), false)
				} else {
					info.paramIn = in(i)
					p.tagInfos = append(p.tagInfos, info)
					recv.assginIn(in(i), true)
				}
			}
		}
		if len(p.tagInfos) == 0 {
			for _, i := range sortedDefaultIn {
				if p.omitIns[i] {
					recv.assginIn(i, false)
					continue
				}
				p.tagInfos = append(p.tagInfos, &tagInfo{
					paramIn:   i,
					paramName: p.structField.Name,
				})
				recv.assginIn(i, true)
			}
		}
		if !recv.hasVd {
			_, recv.hasVd = tagKVs.lookup(b.config.Validator)
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
