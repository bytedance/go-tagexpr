package binding

import (
	"errors"
	"net/http"
	"reflect"
	_ "unsafe"

	"github.com/bytedance/go-tagexpr"
	"github.com/bytedance/go-tagexpr/validator"
	"github.com/henrylee2cn/goutil"
	"github.com/henrylee2cn/goutil/tpack"
)

// Binding
type Binding struct {
	Validator      *validator.Validator
	bindErrFactory func(failField, msg string) error
	recvs          goutil.Map
}

// New creates a binding recvect.
// NOTE:
//  If tagName=='', `api` is used
func New(tagName string) *Binding {
	if tagName == "" {
		tagName = "api"
	}
	return &Binding{
		Validator:      validator.New(tagName).SetErrorFactory(defaultValidatingErrFactory),
		bindErrFactory: defaultBindErrFactory,
		recvs:          goutil.AtomicMap(),
	}
}

var defaultValidatingErrFactory = newDefaultErrorFactory("invalid parameter")
var defaultBindErrFactory = newDefaultErrorFactory("binding failed")

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
	b.Validator.SetErrorFactory(validatingErrFactory)
	return b
}

func (b *Binding) BindAndValidate(req *http.Request, structPointer interface{}) error {
	v, err := b.structValueOf(structPointer)
	if err != nil {
		return err
	}
	hasVd, err := b.bind(req, v)
	if err != nil {
		return err
	}
	if hasVd {
		return b.Validator.Validate(v)
	}
	return nil
}

func (b *Binding) Bind(req *http.Request, structPointer interface{}) error {
	v, err := b.structValueOf(structPointer)
	if err != nil {
		return err
	}
	_, err = b.bind(req, v)
	return err
}

func (b *Binding) structValueOf(structPointer interface{}) (reflect.Value, error) {
	v, ok := structPointer.(reflect.Value)
	if !ok {
		v = reflect.ValueOf(structPointer)
	}
	if v.Kind() != reflect.Ptr {
		return v, b.bindErrFactory("", "structPointer must be a non-nil struct pointer")
	}
	v = derefValue(v)
	if v.Kind() != reflect.Struct || !v.CanAddr() || !v.IsValid() {
		return v, b.bindErrFactory("", "structPointer must be a non-nil struct pointer")
	}
	return v, nil
}

func (b *Binding) getObjOrPrepare(value reflect.Value) (*receiver, error) {
	runtimeTypeID := tpack.From(value).RuntimeTypeID()
	i, ok := b.recvs.Load(runtimeTypeID)
	if ok {
		return i.(*receiver), nil
	}

	expr, err := b.Validator.VM().Run(reflect.New(value.Type()).Elem())
	if err != nil {
		return nil, err
	}
	var recv = &receiver{
		params: make([]*paramInfo, 0, 16),
	}
	var errExprSelector tagexpr.ExprSelector
	var errMsg string

	expr.Range(func(es tagexpr.ExprSelector, eval func() interface{}) bool {
		fieldSelector := es.Path()
		if !expr.Field(fieldSelector, true).CanSet() {
			errMsg = "field cannot be set: " + fieldSelector
			errExprSelector = es
			return false
		}
		var in uint8
		switch es.Name() {
		case validator.MatchExprName:
			recv.hasVd = true
			return true
		case validator.ErrMsgExprName:
			return true
		case "raw_body":
			recv.hasRawBody = true
			in = raw_body
		case "body":
			recv.hasBody = true
			in = body
		case "query":
			recv.hasQuery = true
			in = query
		case "path":
			recv.hasPath = true
			in = path
		case "header":
			in = header
		case "cookie":
			in = cookie
		case "required":
			p := recv.getOrAddParam(fieldSelector)
			p.required = tagexpr.FakeBool(eval())
			return true
		default:
			recv.hasBody = true
			recv.hasAuto = true
			return true
		}
		name, errStr := getParamName(es, eval)
		if errStr != "" {
			errMsg = errStr
			errExprSelector = es
			return false
		}
		p := recv.getOrAddParam(fieldSelector)
		p.name = name
		p.requiredError = errors.New("missing required parameter: " + name)
		p.typeError = errors.New("parameter type does not match binding data: " + name)
		p.cannotError = errors.New("parameter cannot be bound: " + name)
		p.in = in
		return true
	})
	if errMsg != "" {
		return nil, b.bindErrFactory(errExprSelector.String(), errMsg)
	}
	b.recvs.Store(runtimeTypeID, recv)
	return recv, nil
}

func (b *Binding) bind(req *http.Request, value reflect.Value) (hasVd bool, err error) {
	recv, err := b.getObjOrPrepare(value)
	if err != nil {
		return false, err
	}

	expr, err := b.Validator.VM().Run(value)
	if err != nil {
		return false, err
	}

	bodyBytes, err := recv.getBodyBytes(req)
	if err != nil {
		return false, err
	}

	bodyCodec := getBodyCodec(req)
	queryValues := recv.getQuery(req)

	for _, param := range recv.params {
		v, err := param.getField(expr)
		if err != nil {
			return recv.hasVd, err
		}
		if !v.IsValid() {
			continue
		}
		switch param.in {
		case query:
			err = param.bindQuery(v, queryValues)
		case path:
		case header:
			err = param.bindHeader(v, req.Header)
		case cookie:
		case body:
			_ = bodyCodec
		case raw_body:
			err = param.bindRawBody(v, bodyBytes)
		default:
		}
		if err != nil {
			return recv.hasVd, err
		}
	}
	return recv.hasVd, nil
}
