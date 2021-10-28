package gjson

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/henrylee2cn/ameda"
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

	data = []byte(`{
          "M1": {
            "i": 9,
            "j": "*"
          },
          "M2": {
            "k2": {
              "A": "a2",
              "B": 12
            }
          },
          "M3": {
            "k3": {
              "A": "a2",
              "B": "13"
            }
          }
        }`)

	var x3 *X
	err = unmarshal(data, &x3)
	assert.NoError(t, err)
	assert.Equal(t, x, *x3)
}

func TestStruct(t *testing.T) {
	type a struct {
		V int `json:"v"`
	}
	type B struct {
		a
		A2 **a
	}
	type C struct {
		*B `json:"b"`
	}
	type D struct {
		*C `json:","`
		C2 *int
	}
	type E struct {
		D
		K int `json:"k"`
		int
	}
	data := []byte(`{
"k":1,
"C2":null,
"b":{"v":2,"A2":{"v":3}}
}`)
	std := &E{}
	err := json.Unmarshal(data, std)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, std.K)
		assert.Equal(t, 2, std.V)
		assert.Equal(t, 3, (*std.A2).V)
	}
	g := &E{}
	err = unmarshal(data, g)
	assert.NoError(t, err)
	assert.Equal(t, std, g)

	type X struct {
		*X
		Y int
	}
	data2 := []byte(`{"X":{"Y":2}}`)
	std2 := &X{}
	err = json.Unmarshal(data2, std2)
	if assert.NoError(t, err) {
		t.Logf("%#v", std2)
	}
	g2 := &X{}
	err = unmarshal(data2, g2)
	assert.NoError(t, err)
	assert.Equal(t, std2, g2)
}

func TestAliasBUG1(t *testing.T) {
	type DeviceUUID string
	type DeviceUUIDMap map[DeviceUUID]string
	type AttachedMobiles struct {
		AttachedAndroid DeviceUUIDMap `json:"android,omitempty"`
		AttachedIOS     DeviceUUIDMap `json:"ios,omitempty"`
	}
	b, err := json.MarshalIndent(ameda.InitSampleValue(reflect.TypeOf(AttachedMobiles{}), 10).Interface(), "", "  ")
	assert.NoError(t, err)
	var r AttachedMobiles
	err = unmarshal(b, &r)
	assert.NoError(t, err)
	// b, err = json.Marshal(map[float32]int{
	// 	1.0: 4,
	// })
	// assert.NoError(t, err)
	// t.Log(string(b))
}
