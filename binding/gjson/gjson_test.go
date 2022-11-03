package gjson

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"testing"

	"github.com/henrylee2cn/ameda"
	"github.com/stretchr/testify/assert"

	"github.com/bytedance/go-tagexpr/v2/binding/gjson/internal/rt"
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

	err := Unmarshal(data, &x2)
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
	err = Unmarshal(data, &x3)
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
	err = Unmarshal(data, g)
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
	err = Unmarshal(data2, g2)
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
	err = Unmarshal(b, &r)
	assert.NoError(t, err)
	// b, err = json.Marshal(map[float32]int{
	// 	1.0: 4,
	// })
	// assert.NoError(t, err)
	// t.Log(string(b))
}

func TestBingSliceWithObject(t *testing.T) {
	type F struct {
		UID int64
	}
	type foo struct {
		F1 []F `json:"f1"`
		F2 []F `json:"f2"`
	}
	str := `{"f1":{"UID":1},"f2":[{"UID":"2233"}]}`

	obj := foo{}
	err := Unmarshal([]byte(str), &obj)

	assert.NoError(t, err)
	assert.Len(t, obj.F1, 0)
}
func BenchmarkGetFiledInfo(b *testing.B) {
	var types []reflect.Type
	const count = 2000
	for i := 0; i < count; i++ {
		xtype := genStruct(i)

		getFiledInfo(xtype)

		types = append(types, xtype)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i int64
		for pb.Next() {
			getFiledInfo(types[i%count])
			i++
		}
	})
}
func BenchmarkGetFieldInfoByMap(b *testing.B) {
	var types []reflect.Type
	const count = 2000
	for i := 0; i < count; i++ {
		xtype := genStruct(i)

		getFiledInfoWithMap(xtype)

		types = append(types, xtype)
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		var i int64
		for pb.Next() {
			getFiledInfoWithMap(types[i%count])
			i++
		}
	})
}

func genStruct(n int) reflect.Type {
	numOfFields := rand.Intn(50) + 1
	field := make([]reflect.StructField, 0, numOfFields)
	for i := 0; i < numOfFields; i++ {
		field = append(field, reflect.StructField{
			Name:    fmt.Sprintf("F%d_%d", n, i),
			PkgPath: "",
			Type: reflect.TypeOf(struct {
				A int
				B map[int]interface{}
			}{}),
		})
	}
	ot := reflect.StructOf(field)
	return ot
}

var fieldsmu sync.RWMutex
var fields = make(map[uintptr]map[string][]int)

func getFiledInfoWithMap(t reflect.Type) map[string][]int {

	runtimeTypeID := ameda.RuntimeTypeID(t)
	fieldsmu.RLock()
	sf := fields[runtimeTypeID]
	fieldsmu.RUnlock()
	if sf == nil {
		fieldsmu.Lock()
		defer fieldsmu.Unlock()

		d := rt.UnpackType(t)
		sf1, _ := computeTypeInfo(d)
		sf = sf1.(map[string][]int)
		fields[runtimeTypeID] = sf

	}
	return sf
}


// MarshalJSON to output non base64 encoded []byte
func (j ByteSlice) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}

	return json.RawMessage(j).MarshalJSON()
}

// UnmarshalJSON to deserialize []byte
func (j *ByteSlice) UnmarshalJSON(b []byte) error {
	result := json.RawMessage{}
	err := result.UnmarshalJSON(b)
	*j = ByteSlice(result)
	return err
}

type ByteSlice []byte

func TestCustomizedGjsonUnmarshal(t *testing.T) {
	str := `{"h1":{"h2":1}}`
	type F struct {
		H ByteSlice `json:"h1"`
	}

	obj := F{}
	err := Unmarshal([]byte(str), &obj)

	assert.NoError(t, err)
	assert.Equal(t, "{\"h2\":1}", string(obj.H))

	obj2 := F{}
	err = json.Unmarshal([]byte(str), &obj2)
	assert.NoError(t, err)
	assert.Equal(t, "{\"h2\":1}", string(obj2.H))

	assert.Equal(t, obj.H, obj2.H)
}
