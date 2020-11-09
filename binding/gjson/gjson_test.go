package gjson

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	type X struct {
		M1 map[string]interface{}
		M2 map[string]struct {
			A string
			B int
		}
		M3 map[string]*struct {
			A string
			B int
		}
	}
	x := X{
		M1: map[string]interface{}{"i": float64(9), "j": "*"},
		M2: map[string]struct {
			A string
			B int
		}{"k2": {"a2", 12}},
		M3: map[string]*struct {
			A string
			B int
		}{"k3": {"a2", 13}},
	}
	data, _ := json.MarshalIndent(x, "", "  ")
	t.Log(string(data))

	var x2 X
	err := unmarshal(data, &x2)
	assert.NoError(t, err)
	assert.Equal(t, x, x2)
}
