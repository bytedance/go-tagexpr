package validator_test

import (
	"encoding/json"
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
	assert.EqualError(t, vd.Validate(new(T), true), "email format is incorrect\tinvalid parameter: f.g")
}

func TestIssue1(t *testing.T) {
	type MailBox struct {
		Address *string `vd:"email($)"`
		Name    *string
	}
	type EmailMsg struct {
		Recipients       []*MailBox
		RecipientsCc     []*MailBox
		RecipientsBcc    []*MailBox
		Subject          *string
		Content          *string
		AttachmentIDList []string
		ReplyTo          *string
		Params           map[string]string
		FromEmailAddress *string
		FromEmailName    *string
	}
	type EmailTaskInfo struct {
		Msg         *EmailMsg
		StartTimeMS *int64
		LogTag      *string
	}
	type BatchCreateEmailTaskRequest struct {
		InfoList []*EmailTaskInfo
	}
	var invalid = "invalid email"
	req := &BatchCreateEmailTaskRequest{
		InfoList: []*EmailTaskInfo{
			{
				Msg: &EmailMsg{
					Recipients: []*MailBox{
						{
							Address: &invalid,
						},
					},
				},
			},
		},
	}
	assert.EqualError(t, vd.Validate(req, true), "email format is incorrect")
}

func TestIssue2(t *testing.T) {
	type a struct {
		m map[string]interface{}
	}
	A := &a{
		m: map[string]interface{}{
			"1": 1,
			"2": nil,
		},
	}
	v := vd.New("vd")
	assert.NoError(t, v.Validate(A))
}

func TestIssue3(t *testing.T) {
	type C struct {
		Id    string
		Index int32 `vd:"$==1"`
	}
	type A struct {
		F1 *C
		F2 *C
	}
	a := &A{
		F1: &C{
			Id:    "test",
			Index: 1,
		},
	}
	v := vd.New("vd")
	assert.NoError(t, v.Validate(a))
}

func TestIssue4(t *testing.T) {
	type C struct {
		Index  *int32 `vd:"@:$!=nil;msg:'index is nil'"`
		Index2 *int32 `vd:"$!=nil"`
		Index3 *int32 `vd:"$!=nil"`
	}
	type A struct {
		F1 *C
		F2 map[string]*C
		F3 []*C
	}
	v := vd.New("vd")

	a := &A{}
	assert.NoError(t, v.Validate(a))

	a = &A{F1: new(C)}
	assert.EqualError(t, v.Validate(a), "index is nil")

	a = &A{F2: map[string]*C{"": &C{Index: new(int32)}}}
	assert.EqualError(t, v.Validate(a), "invalid parameter: F2{}.Index2")

	a = &A{F3: []*C{{Index: new(int32)}}}
	assert.EqualError(t, v.Validate(a), "invalid parameter: F3[0].Index2")

	type B struct {
		F1 *C `vd:"$!=nil"`
		F2 *C
	}
	b := &B{}
	assert.EqualError(t, v.Validate(b), "invalid parameter: F1")

	type D struct {
		F1 *C
		F2 *C
	}

	type E struct {
		D []*D
	}
	b.F1 = new(C)
	e := &E{D: []*D{nil}}
	assert.NoError(t, v.Validate(e))
}

func TestIssue5(t *testing.T) {
	type SubSheet struct {
	}
	type CopySheet struct {
		Source      *SubSheet `json:"source" vd:"$!=nil"`
		Destination *SubSheet `json:"destination" vd:"$!=nil"`
	}
	type UpdateSheetsRequest struct {
		CopySheet *CopySheet `json:"copySheet"`
	}
	type BatchUpdateSheetRequestArg struct {
		Requests []*UpdateSheetsRequest `json:"requests"`
	}
	b := `{"requests": [{}]}`
	var data BatchUpdateSheetRequestArg
	err := json.Unmarshal([]byte(b), &data)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(data.Requests))
	assert.Nil(t, data.Requests[0].CopySheet)
	v := vd.New("vd")
	assert.NoError(t, v.Validate(data))
}

func TestIn(t *testing.T) {
	type S string
	type I int16
	type T struct {
		A S `vd:"in($,'a','b','c')"`
		B I `vd:"in($,1,2.0,3)"`
	}
	v := vd.New("vd")
	data := T{}
	err := v.Validate(data)
	assert.EqualError(t, err, "[a b c] range exceeded")
	data.A = "b"
	err = v.Validate(data)
	assert.EqualError(t, err, "[1 2 3] range exceeded")
	data.B = 2
	err = v.Validate(data)
	assert.NoError(t, err)

	type T2 struct {
		C string `vd:"in($)"`
	}
	data2 := T2{}
	err = v.Validate(data2)
	assert.EqualError(t, err, "input parameters of the in function are at least two")

	type T3 struct {
		C string `vd:"in($,1)"`
	}
	data3 := T3{}
	err = v.Validate(data3)
	assert.EqualError(t, err, "[1] range exceeded")
}
