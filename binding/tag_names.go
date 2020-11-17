package binding

import (
	"net/textproto"
	"reflect"
	"sort"
	"strings"

	"github.com/henrylee2cn/goutil"
)

const (
	tagRequired         = "required"
	tagRequired2        = "req"
	defaultTagPath      = "path"
	defaultTagQuery     = "query"
	defaultTagHeader    = "header"
	defaultTagCookie    = "cookie"
	defaultTagRawbody   = "raw_body"
	defaultTagForm      = "form"
	defaultTagValidator = "vd"
	tagProtobuf         = "protobuf"
	tagJSON             = "json"
	tagDefault          = "default"
)

// Config the struct tag naming and so on
type Config struct {
	// LooseZeroMode if set to true,
	// the empty string request parameter is bound to the zero value of parameter.
	// NOTE: Suitable for these parameter types: query/header/cookie/form .
	LooseZeroMode bool
	// PathParam use 'path' by default when empty
	PathParam string
	// Query use 'query' by default when empty
	Query string
	// Header use 'header' by default when empty
	Header string
	// Cookie use 'cookie' by default when empty
	Cookie string
	// RawBody use 'raw' by default when empty
	RawBody string
	// FormBody use 'form' by default when empty
	FormBody string
	// Validator use 'vd' by default when empty
	Validator string
	// protobufBody use 'protobuf' by default when empty
	protobufBody string
	// jsonBody use 'json' by default when empty
	jsonBody string
	// defaultVal use 'default' by default when empty
	defaultVal string

	list []string
}

func (t *Config) init() {
	t.list = []string{
		goutil.InitAndGetString(&t.PathParam, defaultTagPath),
		goutil.InitAndGetString(&t.Query, defaultTagQuery),
		goutil.InitAndGetString(&t.Header, defaultTagHeader),
		goutil.InitAndGetString(&t.Cookie, defaultTagCookie),
		goutil.InitAndGetString(&t.RawBody, defaultTagRawbody),
		goutil.InitAndGetString(&t.FormBody, defaultTagForm),
		goutil.InitAndGetString(&t.Validator, defaultTagValidator),
		goutil.InitAndGetString(&t.protobufBody, tagProtobuf),
		goutil.InitAndGetString(&t.jsonBody, tagJSON),
		goutil.InitAndGetString(&t.defaultVal, tagDefault),
	}
}

func (t *Config) parse(field reflect.StructField) tagKVs {
	tag := field.Tag
	fieldName := field.Name

	kvs := make(tagKVs, 0, len(t.list))
	s := string(tag)

	for _, name := range t.list {
		value, ok := tag.Lookup(name)
		if !ok {
			continue
		}
		if name != t.defaultVal && value != "-" {
			value = strings.Replace(strings.TrimSpace(value), " ", "", -1)
			value = strings.Replace(value, "\t", "", -1)
			if name == t.RawBody {
				info := newTagInfo(value, false)
				if info.required || info.paramName == tagRequired {
					value = "," + tagRequired
				}
			} else if value == "" {
				value = fieldName
			} else if value == ","+tagRequired {
				value = fieldName + value
			}
		}
		kvs = append(kvs, &tagKV{name: name, value: value, pos: strings.Index(s, name)})
	}
	sort.Sort(kvs)
	return kvs
}

type tagKV struct {
	name  string
	value string
	pos   int
}

type tagInfo struct {
	paramIn   in
	paramName string
	required  bool
	namePath  string

	requiredError, typeError, cannotError, contentTypeError error
}

func (t *tagKV) toInfo(isHeader bool) *tagInfo {
	return newTagInfo(t.value, isHeader)
}

func newTagInfo(value string, isHeader bool) *tagInfo {
	info := new(tagInfo)
	for i, v := range strings.Split(value, ",") {
		v = strings.TrimSpace(v)
		if i == 0 {
			info.paramName = v
		} else {
			if v == tagRequired || v == tagRequired2 {
				info.required = true
			}
		}
	}
	if isHeader {
		info.paramName = textproto.CanonicalMIMEHeaderKey(info.paramName)
	}
	return info
}

type tagKVs []*tagKV

// Len is the number of elements in the collection.
func (a tagKVs) Len() int {
	return len(a)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (a tagKVs) Less(i, j int) bool {
	return a[i].pos < a[j].pos
}

// Swap swaps the elements with indexes i and j.
func (a tagKVs) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a tagKVs) lookup(name string) (string, bool) {
	for _, v := range a {
		if v.name == name {
			return v.value, true
		}
	}
	return "", false
}
