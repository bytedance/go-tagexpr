package binding

import (
	jsonpkg "encoding/json"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/henrylee2cn/ameda"
	"github.com/henrylee2cn/goutil"

	"github.com/bytedance/go-tagexpr/v2"
	"github.com/bytedance/go-tagexpr/v2/validator"
)

// Binding binding and verification tool for http request
type Binding struct {
	vd             *validator.Validator
	recvs          map[uintptr]*receiver
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
		recvs:  make(map[uintptr]*receiver, 1024),
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
	return b.IBindAndValidate(recvPointer, wrapRequest(req), pathParams)
}

// Bind binds the request parameters.
func (b *Binding) Bind(recvPointer interface{}, req *http.Request, pathParams PathParams) error {
	return b.IBind(recvPointer, wrapRequest(req), pathParams)
}

// IBindAndValidate binds the request parameters and validates them if needed.
func (b *Binding) IBindAndValidate(recvPointer interface{}, req Request, pathParams PathParams) error {
	v, hasVd, err := b.bind(recvPointer, req, pathParams)
	if err != nil {
		return err
	}
	if hasVd {
		return b.vd.Validate(v)
	}
	return nil
}

// IBind binds the request parameters.
func (b *Binding) IBind(recvPointer interface{}, req Request, pathParams PathParams) error {
	_, _, err := b.bind(recvPointer, req, pathParams)
	return err
}

// Validate validates whether the fields of value is valid.
func (b *Binding) Validate(value interface{}) error {
	return b.vd.Validate(value)
}

func (b *Binding) bind(pointer interface{}, req Request, pathParams PathParams) (elemValue reflect.Value, hasVd bool, err error) {
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

func (b *Binding) bindNonstruct(pointer interface{}, _ reflect.Value, req Request, _ PathParams) (hasVd bool, err error) {
	bodyCodec := getBodyCodec(req)
	switch bodyCodec {
	case bodyJSON:
		hasVd = true
		bodyBytes, err := req.GetBody()
		if err != nil {
			return hasVd, err
		}
		err = bindJSON(pointer, bodyBytes)
	case bodyProtobuf:
		hasVd = true
		bodyBytes, err := req.GetBody()
		if err != nil {
			return hasVd, err
		}
		err = bindProtobuf(pointer, bodyBytes)
	case bodyForm:
		postForm, err := req.GetPostForm()
		if err != nil {
			return false, err
		}
		b, _ := jsonpkg.Marshal(postForm)
		err = jsonpkg.Unmarshal(b, pointer)
	default:
		// query and form
		form, err := req.GetForm()
		if err != nil {
			return false, err
		}
		b, _ := jsonpkg.Marshal(form)
		err = jsonpkg.Unmarshal(b, pointer)
	}
	return
}

func (b *Binding) bindStruct(structPointer interface{}, structValue reflect.Value, req Request, pathParams PathParams) (hasVd bool, err error) {
	recv, err := b.getOrPrepareReceiver(structValue)
	if err != nil {
		return
	}

	expr, err := b.vd.VM().Run(structValue)
	if err != nil {
		return
	}

	bodyCodec, bodyBytes, err := recv.getBodyInfo(req)
	if len(bodyBytes) > 0 {
		err = recv.prebindBody(structPointer, structValue, bodyCodec, bodyBytes)
	}
	if err != nil {
		return
	}
	bodyString := ameda.UnsafeBytesToString(bodyBytes)
	postForm, err := req.GetPostForm()
	if err != nil {
		return
	}
	var fileHeaders map[string][]*multipart.FileHeader
	if _req, ok := req.(requestWithFileHeader); ok {
		fileHeaders, err = _req.GetFileHeaders()
		if err != nil {
			return
		}
	}
	queryValues := recv.getQuery(req)
	cookies := recv.getCookies(req)

	for _, param := range recv.params {
		for i, info := range param.tagInfos {
			var found bool
			switch info.paramIn {
			case raw_body:
				err = param.bindRawBody(info, expr, bodyBytes)
				found = err == nil
			case path:
				found, err = param.bindPath(info, expr, pathParams)
			case query:
				found, err = param.bindQuery(info, expr, queryValues)
			case cookie:
				found, err = param.bindCookie(info, expr, cookies)
			case header:
				found, err = param.bindHeader(info, expr, req.GetHeader())
			case form, json, protobuf:
				if info.paramIn == in(bodyCodec) {
					found, err = param.bindOrRequireBody(info, expr, bodyCodec, bodyString, postForm, fileHeaders,
						recv.hasDefaultVal)
				} else if info.required {
					found = false
					err = info.requiredError
				}
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
		v = ameda.DereferencePtrValue(v)
		if v.IsValid() && v.CanAddr() {
			return v, nil
		}
	}
	return v, b.bindErrFactory("", "receiver must be a non-nil pointer")
}

func (b *Binding) getOrPrepareReceiver(value reflect.Value) (*receiver, error) {
	runtimeTypeID := ameda.ValueFrom(value).RuntimeTypeID()
	b.lock.RLock()
	recv, ok := b.recvs[runtimeTypeID]
	b.lock.RUnlock()
	if ok {
		return recv, nil
	}
	t := value.Type()
	expr, err := b.vd.VM().Run(reflect.New(t).Elem())
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
				tagInfos[paramIn] = tagKV.toInfo(paramIn == header)
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
		switch len(p.tagInfos) {
		case 0:
			var canDefault = true
			for s := range fieldsWithValidTag {
				if strings.HasPrefix(fs, s) {
					canDefault = false
					break
				}
			}
			if canDefault {
				if !goutil.IsExportedName(p.structField.Name) {
					canDefault = false
				}
			}
			// Supports the default binding order when there is no valid tag in the superior field of the exportable field
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
		case 1:
			if p.tagInfos[0].paramIn == default_val {
				last := p.tagInfos[0]
				p.tagInfos = make([]*tagInfo, 0, len(sortedDefaultIn)+1)
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
				p.tagInfos = append(p.tagInfos, last)
			}
			fallthrough
		default:
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
	if !recv.hasVd {
		recv.hasVd, _ = b.findVdTag(ameda.DereferenceType(t), false, 20, map[reflect.Type]bool{})
	}
	recv.initParams()

	b.lock.Lock()
	b.recvs[runtimeTypeID] = recv
	b.lock.Unlock()

	return recv, nil
}

func (b *Binding) findVdTag(t reflect.Type, inMapOrSlice bool, depth int, exist map[reflect.Type]bool) (hasVd bool, err error) {
	if depth <= 0 || exist[t] {
		return
	}
	depth--
	switch t.Kind() {
	case reflect.Struct:
		exist[t] = true
		for i := t.NumField() - 1; i >= 0; i-- {
			field := t.Field(i)
			if inMapOrSlice {
				tagKVs := b.config.parse(field)
				for _, tagKV := range tagKVs {
					if tagKV.name == b.config.Validator {
						return true, nil
					}
				}
			}
			hasVd, _ = b.findVdTag(ameda.DereferenceType(field.Type), inMapOrSlice, depth, exist)
			if hasVd {
				return true, nil
			}
		}
		return false, nil
	case reflect.Slice, reflect.Array, reflect.Map:
		return b.findVdTag(ameda.DereferenceType(t.Elem()), true, depth, exist)
	default:
		return false, nil
	}
}
