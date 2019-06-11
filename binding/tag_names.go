package binding

import (
	"reflect"
	"sort"
	"strings"
)

const (
	tagRequired         = "required"
	defaultTagPath      = "path"
	defaultTagQuery     = "query"
	defaultTagHeader    = "header"
	defaultTagCookie    = "cookie"
	defaultTagRawbody   = "rawbody"
	defaultTagForm      = "form"
	defaultTagValidator = "vd"
	tagProtobuf         = "protobuf"
	tagJSON             = "json"
)

// TagNames struct tag naming
type TagNames struct {
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

	list []string
}

func (t *TagNames) init() {
	t.list = []string{
		getAndSet(&t.PathParam, defaultTagPath),
		getAndSet(&t.Query, defaultTagQuery),
		getAndSet(&t.Header, defaultTagHeader),
		getAndSet(&t.Cookie, defaultTagCookie),
		getAndSet(&t.RawBody, defaultTagRawbody),
		getAndSet(&t.FormBody, defaultTagForm),
		getAndSet(&t.Validator, defaultTagValidator),
		getAndSet(&t.protobufBody, tagProtobuf),
		getAndSet(&t.jsonBody, tagJSON),
	}
}

func getAndSet(s *string, def string) string {
	if *s == "" {
		*s = def
	}
	return *s
}

func (t *TagNames) parse(field reflect.StructField) tagKVs {
	tag := field.Tag
	fieldName := field.Name

	kvs := make(tagKVs, 0, len(t.list))
	s := string(tag)

	for _, name := range t.list {
		value, ok := tag.Lookup(name)
		if !ok {
			continue
		}
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
			if v == tagRequired {
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
