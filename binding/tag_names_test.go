package binding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultSplitTag(t *testing.T) {
	var cases = []struct {
		desc     string
		input    string
		expected *tagInfo
	}{
		{
			desc:     "default",
			input:    "a",
			expected: &tagInfo{paramName: "a"},
		},
		{
			desc:     "default required",
			input:    "a,required",
			expected: &tagInfo{paramName: "a", required: true},
		},
		{
			desc:     "slice",
			input:    "[1,2,3]",
			expected: &tagInfo{paramName: "[1,2,3]"},
		},
		{
			desc:     "slice required",
			input:    "[1,2,3],req",
			expected: &tagInfo{paramName: "[1,2,3]", required: true},
		},
		{
			desc:     "map",
			input:    "{col1:a,col2:b},req",
			expected: &tagInfo{paramName: "{col1:a,col2:b}", required: true},
		},
		{
			desc:     "invalid map",
			input:    "{col1:a,col2}",
			expected: &tagInfo{paramName: "{col1:a"},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.expected, defaultSplitTag(c.input))
		})
	}
}
