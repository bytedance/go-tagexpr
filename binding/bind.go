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

// Binding binding and verification tool for http request
type Binding struct {
	vd             *validator.Validator
	bindErrFactory func(failField, msg string) error
	recvs          goutil.Map
}

// New creates a binding tool.
// NOTE:
//  If tagName=='', `api` is used
func New(tagName string) *Binding {
	if tagName == "" {
		tagName = "api"
	}
	return &Binding{
		vd:             validator.New(tagName).SetErrorFactory(defaultValidatingErrFactory),
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
	b.vd.SetErrorFactory(validatingErrFactory)
	return b
}

// BindAndValidate binds the request parameters and validates them if needed.
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
		return b.vd.Validate(v)
	}
	return nil
}

// Bind binds the request parameters.
func (b *Binding) Bind(req *http.Request, structPointer interface{}) error {
	v, err := b.structValueOf(structPointer)
	if err != nil {
		return err
	}
	_, err = b.bind(req, v)
	return err
}

// Validate validates whether the fields of v is valid.
func (b *Binding) Validate(value interface{}) error {
	return b.vd.Validate(value)
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

	expr, err := b.vd.VM().Run(reflect.New(value.Type()).Elem())
	if err != nil {
		return nil, err
	}
	var recv = &receiver{
		params: make([]*paramInfo, 0, 16),
	}
	var errExprSelector tagexpr.ExprSelector
	var errMsg string

	expr.RangeFields(func(fh *tagexpr.FieldHandler) bool {
		fieldSelector := fh.StringSelector()

		if !fh.Value(true).CanSet() {
			errMsg = "field cannot be set: " + fieldSelector
			errExprSelector = tagexpr.ExprSelector(fieldSelector)
			return false
		}

		p := recv.getOrAddParam(fh)
		in := auto
		name := fh.FieldSelector().Name()

	L:
		for es, eval := range fh.EvalFuncs() {

			switch es.Name() {
			case validator.MatchExprName:
				recv.hasVd = true
				continue L
			case validator.ErrMsgExprName:
				continue L

			case "required":
				p.required = tagexpr.FakeBool(eval())
				continue L

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
				recv.hasCookie = true
				in = cookie

			default:
				continue L
			}

			name, errMsg = getParamName(eval, name)
			if errMsg != "" {
				errExprSelector = es
				return false
			}
		}

		if in == auto {
			recv.hasBody = true
			recv.hasAuto = true
		}
		p.in = in
		p.name = name
		p.requiredError = errors.New("missing required parameter: " + name)
		p.typeError = errors.New("parameter type does not match binding data: " + name)
		p.cannotError = errors.New("parameter cannot be bound: " + name)
		return true
	})

	if errMsg != "" {
		return nil, b.bindErrFactory(errExprSelector.String(), errMsg)
	}

	recv.combNamePath()

	b.recvs.Store(runtimeTypeID, recv)
	return recv, nil
}

func (b *Binding) bind(req *http.Request, value reflect.Value) (hasVd bool, err error) {
	recv, err := b.getObjOrPrepare(value)
	if err != nil {
		return false, err
	}

	expr, err := b.vd.VM().Run(value)
	if err != nil {
		return false, err
	}

	bodyCodec := recv.getBodyCodec(req)

	bodyBytes, err := recv.getBodyBytes(req, bodyCodec == jsonBody)
	if err != nil {
		return false, err
	}

	postForm, err := recv.getPostForm(req, bodyCodec == formBody)
	if err != nil {
		return false, err
	}

	queryValues := recv.getQuery(req)
	cookies := recv.getCookies(req)

	for _, param := range recv.params {
		switch param.in {
		case query:
			_, err = param.bindQuery(expr, queryValues)
		case path:
		case header:
			_, err = param.bindHeader(expr, req.Header)
		case cookie:
			err = param.bindCookie(expr, cookies)
		case body:
			_, err = param.bindBody(expr, bodyCodec, postForm, bodyBytes)
		case raw_body:
			err = param.bindRawBody(expr, bodyBytes)
		default:
			var found bool
			found, err = param.bindBody(expr, bodyCodec, postForm, bodyBytes)
			if !found {
				_, err = param.bindQuery(expr, queryValues)
			}
		}
		if err != nil {
			return recv.hasVd, err
		}
	}
	return recv.hasVd, nil
}
