package binding

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnescape(t *testing.T) {
	type testStruct struct {
		A string
		B int
	}

	nonEscaped := "unchanged"
	escaped := fmt.Sprintf("escaped:%s", specialChar)
	unescaped := "escaped:\""

	cases := []struct {
		desc   string
		input  interface{}
		output interface{}
	}{
		{
			desc:   "string",
			input:  &nonEscaped,
			output: &nonEscaped,
		},
		{
			desc:   "string escaped",
			input:  &escaped,
			output: &unescaped,
		},
		{
			desc: "struct escaped",
			input: &testStruct{
				A: fmt.Sprintf("%s - %s", specialChar, specialChar),
				B: 123,
			},

			output: &testStruct{
				A: "\" - \"",
				B: 123,
			},
		},
		{
			desc: "ptr to map escaped",
			input: &map[string]interface{}{
				"a":                                fmt.Sprintf("%s - %s", specialChar, specialChar),
				fmt.Sprintf("%s key", specialChar): 123,
			},

			output: &map[string]interface{}{
				"a":      "\" - \"",
				"\" key": 123,
			},
		},
		{
			desc: "map escaped",
			input: map[string]interface{}{
				"a":                                fmt.Sprintf("%s - %s", specialChar, specialChar),
				fmt.Sprintf("%s key", specialChar): 123,
			},

			output: map[string]interface{}{
				"a":      "\" - \"",
				"\" key": 123,
			},
		},
		{
			desc:   "slice escaped",
			input:  &[]interface{}{"a", fmt.Sprintf("%s - %s", specialChar, specialChar)},
			output: &[]interface{}{"a", "\" - \""},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.output, unescape(reflect.ValueOf(c.input)).Interface())
		})
	}

}
