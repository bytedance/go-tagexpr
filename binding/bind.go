package binding

import (
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/henrylee2cn/ameda"
	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/tpack"
	jsonpkg "github.com/json-iterator/go"

	"github.com/bytedance/go-tagexpr"
	"github.com/bytedance/go-tagexpr/validator"
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
func (b *Binding) BindAndValidate(recvPointer interface{}, req *http.Request, pathParams PathParams) error {
	v, hasVd, err := b.bind(recvPointer, req, pathParams)
	if err != nil {
		return err
	}
	if hasVd {
		return b.vd.Validate(v)
	}
	return nil
}

// Bind binds the request parameters.
func (b *Binding) Bind(recvPointer interface{}, req *http.Request, pathParams PathParams) error {
	_, _, err := b.bind(recvPointer, req, pathParams)
	return err
}

// Validate validates whether the fields of value is valid.
func (b *Binding) Validate(value interface{}) error {
	return b.vd.Validate(value)
}

func (b *Binding) bind(pointer interface{}, req *http.Request, pathParams PathParams) (elemValue reflect.Value, hasVd bool, err error) {
	elemValue, err = b.receiverValueOf(pointer)
	if err != nil {
		return
	}
	if elemValue.Kind() == reflect.Struct {
		hasVd, err = b.bindStruct(pointer, elemValue, req, pathParams)
	} else {
		hasVd, err = b.bindNonstruct(pointer, elemValue, req, pathParams)
	}
	return
}

func (b *Binding) bindNonstruct(pointer interface{}, _ reflect.Value, req *http.Request, _ PathParams) (hasVd bool, err error) {
	bodyCodec := getBodyCodec(req)
	var bodyBytes []byte
	switch bodyCodec {
	case bodyJSON:
		hasVd = true
		bodyBytes, err = getBody(req, bodyCodec)
		if err == nil {
			err = bindJSON(pointer, bodyBytes)
		}
	case bodyProtobuf:
		hasVd = true
		bodyBytes, err = getBody(req, bodyCodec)
		if err == nil {
			err = bindProtobuf(pointer, bodyBytes)
		}
	case bodyForm:
		bodyBytes, err = getBody(req, bodyCodec)
		if err == nil {
			b, _ := jsonpkg.Marshal(req.PostForm)
			err = jsonpkg.Unmarshal(b, pointer)
		}
	default:
		// query and form
		b, _ := jsonpkg.Marshal(req.Form)
		err = jsonpkg.Unmarshal(b, pointer)
	}
	return
}

func (b *Binding) bindStruct(structPointer interface{}, structValue reflect.Value, req *http.Request, pathParams PathParams) (hasVd bool, err error) {
	recv, err := b.getOrPrepareReceiver(structValue)
	if err != nil {
		return
	}

	expr, err := b.vd.VM().Run(structValue)
	if err != nil {
		return
	}

	bodyCodec, bodyBytes, err := recv.getBodyInfo(req)
	if err == nil {
		err = recv.prebindBody(structPointer, structValue, bodyCodec, bodyBytes)
	}
	if err != nil {
		return
	}
	bodyString := ameda.UnsafeBytesToString(bodyBytes)
	postForm := req.PostForm
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
			case default_val:
				found, err = param.bindDefaultVal(expr, param.defaultVal)
			}
			if found && err == nil {
				break
			}
			if (found || i == len(param.tagInfos)-1) && err != nil {
				return recv.hasVd, err
			}
		}
	}
	return recv.hasVd, nil
}

func (b *Binding) receiverValueOf(receiver interface{}) (reflect.Value, error) {
	v := reflect.ValueOf(receiver)
	if v.Kind() == reflect.Ptr {
		v = goutil.DereferencePtrValue(v)
		if v.IsValid() && v.CanAddr() {
			return v, nil
		}
	}
	return v, b.bindErrFactory("", "receiver must be a non-nil pointer")
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
	var fieldsWithValidTag = make(map[string]bool)
	expr.RangeFields(func(fh *tagexpr.FieldHandler) bool {
		if !fh.Value(true).CanSet() {
			selector := fh.StringSelector()
			errMsg = "field cannot be set: " + selector
			errExprSelector = tagexpr.ExprSelector(selector)
			return true
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
			case b.config.defaultVal:
				paramIn = default_val
			default:
				continue L
			}
			if paramIn == default_val {
				tagInfos[paramIn] = &tagInfo{paramIn: default_val, paramName: tagKV.value}
			} else {
				tagInfos[paramIn] = tagKV.defaultSplit()
			}
		}

		for i, info := range tagInfos {
			if info != nil {
				if info.paramIn != default_val && info.paramName == "-" {
					p.omitIns[in(i)] = true
					recv.assginIn(in(i), false)
				} else {
					info.paramIn = in(i)
					p.tagInfos = append(p.tagInfos, info)
					recv.assginIn(in(i), true)
				}
			}
		}
		fs := string(fh.FieldSelector())
		if len(p.tagInfos) == 0 {
			var canDefault = true
			for s := range fieldsWithValidTag {
				if strings.HasPrefix(fs, s) {
					canDefault = false
					break
				}
			}
			// Support default binding order when there is no valid tag in the superior fields
			if canDefault {
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
		} else {
			fieldsWithValidTag[fs+tagexpr.FieldSeparator] = true
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
