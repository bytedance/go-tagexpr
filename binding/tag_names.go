package binding

import (
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
				info := defaultSplitTag(value)
				if info.required || info.paramName == tagRequired {
					value = "," + tagRequired
				}
			} else if value == "" {
				value = fieldName
			} else if value == ","+tagRequired {
				value = fieldName + value
			}
		}
		// make sure header key style
		if name == t.Header {
			b := []byte(value)
			normalizeHeaderKey(b)
			value = string(b)
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

func (t *tagKV) defaultSplit() *tagInfo {
	return defaultSplitTag(t.value)
}

func defaultSplitTag(value string) *tagInfo {
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

func normalizeHeaderKey(b []byte) {
	n := len(b)
	if n == 0 {
		return
	}
	b[0] = toUpperTable[b[0]]
	for i := 1; i < n; i++ {
		p := &b[i]
		if *p == '-' {
			i++
			if i < n {
				b[i] = toUpperTable[b[i]]
			}
			continue
		}
		*p = toLowerTable[*p]
	}
}

const toLowerTable = "\x00\x01\x02\x03\x04\x05\x06\a\b\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f !\"#$%&'()*+,-./0123456789:;<=>?@abcdefghijklmnopqrstuvwxyz[\\]^_`abcdefghijklmnopqrstuvwxyz{|}~\u007f\x80\x81\x82\x83\x84\x85\x86\x87\x88\x89\x8a\x8b\x8c\x8d\x8e\x8f\x90\x91\x92\x93\x94\x95\x96\x97\x98\x99\x9a\x9b\x9c\x9d\x9e\x9f\xa0\xa1\xa2\xa3\xa4\xa5\xa6\xa7\xa8\xa9\xaa\xab\xac\xad\xae\xaf\xb0\xb1\xb2\xb3\xb4\xb5\xb6\xb7\xb8\xb9\xba\xbb\xbc\xbd\xbe\xbf\xc0\xc1\xc2\xc3\xc4\xc5\xc6\xc7\xc8\xc9\xca\xcb\xcc\xcd\xce\xcf\xd0\xd1\xd2\xd3\xd4\xd5\xd6\xd7\xd8\xd9\xda\xdb\xdc\xdd\xde\xdf\xe0\xe1\xe2\xe3\xe4\xe5\xe6\xe7\xe8\xe9\xea\xeb\xec\xed\xee\xef\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xfe\xff"
const toUpperTable = "\x00\x01\x02\x03\x04\x05\x06\a\b\t\n\v\f\r\x0e\x0f\x10\x11\x12\x13\x14\x15\x16\x17\x18\x19\x1a\x1b\x1c\x1d\x1e\x1f !\"#$%&'()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\]^_`ABCDEFGHIJKLMNOPQRSTUVWXYZ{|}~\u007f\x80\x81\x82\x83\x84\x85\x86\x87\x88\x89\x8a\x8b\x8c\x8d\x8e\x8f\x90\x91\x92\x93\x94\x95\x96\x97\x98\x99\x9a\x9b\x9c\x9d\x9e\x9f\xa0\xa1\xa2\xa3\xa4\xa5\xa6\xa7\xa8\xa9\xaa\xab\xac\xad\xae\xaf\xb0\xb1\xb2\xb3\xb4\xb5\xb6\xb7\xb8\xb9\xba\xbb\xbc\xbd\xbe\xbf\xc0\xc1\xc2\xc3\xc4\xc5\xc6\xc7\xc8\xc9\xca\xcb\xcc\xcd\xce\xcf\xd0\xd1\xd2\xd3\xd4\xd5\xd6\xd7\xd8\xd9\xda\xdb\xdc\xdd\xde\xdf\xe0\xe1\xe2\xe3\xe4\xe5\xe6\xe7\xe8\xe9\xea\xeb\xec\xed\xee\xef\xf0\xf1\xf2\xf3\xf4\xf5\xf6\xf7\xf8\xf9\xfa\xfb\xfc\xfd\xfe\xff"
