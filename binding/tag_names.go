package binding

import (
	"reflect"
	"sort"
	"strings"
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
	setDefault(&t.PathParam, "path")
	setDefault(&t.Query, "query")
	setDefault(&t.Header, "header")
	setDefault(&t.Cookie, "cookie")
	setDefault(&t.RawBody, "raw_body")
	setDefault(&t.FormBody, "form")
	setDefault(&t.Validator, "vd")
	setDefault(&t.protobufBody, "protobuf")
	setDefault(&t.jsonBody, "json")
	t.list = []string{
		t.PathParam,
		t.Query,
		t.Header,
		t.Cookie,
		t.RawBody,
		t.FormBody,
		t.Validator,
		t.protobufBody,
		t.jsonBody,
	}
}

func setDefault(s *string, def string) {
	if *s == "" {
		*s = def
	}
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
			if _, required := defaultSplitTag(value); required {
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

const tagRequired = "required"

func (t *tagKV) defaultSplit() (paramName string, required bool) {
	return defaultSplitTag(t.value)
}

func defaultSplitTag(value string) (paramName string, required bool) {
	for i, v := range strings.Split(value, ",") {
		v = strings.TrimSpace(v)
		if i == 0 {
			paramName = v
		} else {
			if v == tagRequired {
				required = true
			}
		}
	}
	return paramName, required
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
