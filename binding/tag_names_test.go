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
			desc:     "default empty",
			input:    "",
			expected: &tagInfo{},
		},
		{
			desc:     "default",
			input:    "a,required",
			expected: &tagInfo{paramName: "a", required: true},
		},
	}

	for _, c := range cases {
		t.Run(c.desc, func(t *testing.T) {
			assert.Equal(t, c.expected, newTagInfo(c.input, false))
		})
	}
}
