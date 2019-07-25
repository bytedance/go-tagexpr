package validator_test

import (
	"testing"

	vd "github.com/bytedance/go-tagexpr/validator"
	"github.com/stretchr/testify/assert"
)

func TestNil(t *testing.T) {
	type F struct {
		f struct {
			g int `vd:"$%3==1"`
		}
	}
	assert.EqualError(t, vd.Validate((*F)(nil)), "unsupport data: nil")
}

func TestAll(t *testing.T) {
	type T struct {
		a string `vd:"email($)"`
		f struct {
			g int `vd:"$%3==1"`
		}
	}
	assert.EqualError(t, vd.Validate(new(T), true), "invalid parameter: a\tinvalid parameter: f.g")
}
