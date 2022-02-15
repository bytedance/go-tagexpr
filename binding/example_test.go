package binding_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/henrylee2cn/goutil/httpbody"

	"github.com/bytedance/go-tagexpr/v2/binding"
)

func Example() {
	type InfoRequest struct {
		Name          string   `path:"name"`
		Year          []int    `query:"year"`
		Pages         []uint64 `query:"pages"`
		Email         *string  `json:"email" vd:"email($)"`
		Friendly      bool     `json:"friendly"`
		Pie           float32  `json:"pie,required"`
		Hobby         []string `json:",required"`
		BodyNotFound  *int     `json:"BodyNotFound"`
		Authorization string   `header:"Authorization,required" vd:"$=='Basic 123456'"`
		userIdHeader  string   `header:"x-user_ID,required"`
		SessionID     string   `cookie:"sessionid,required"`
		AutoBody      string
		AutoNotFound  *string
		TimeRFC3339   time.Time `query:"t"`
	}

	binding.MustRegTypeUnmarshal(reflect.TypeOf([]uint64{}), func(v string, emptyAsZero bool) (reflect.Value, error) {
		if v == "" && emptyAsZero {
			return reflect.ValueOf([]uint64{}), nil
		}

		ss := strings.Split(v, ",")
		t := make([]uint64, 0, len(ss))

		for _, s := range ss {
			i, err := strconv.ParseUint(s, 10, 64)
			if err != nil {
				return reflect.ValueOf([]uint64{}), err
			}
			t = append(t, i)
		}

		return reflect.ValueOf(t), nil
	})

	args := new(InfoRequest)
	binder := binding.New(nil)
	err := binder.BindAndValidate(args, requestExample(), new(testPathParams))

	fmt.Println("bind and validate result:")

	fmt.Printf("error: %v\n", err)

	b, _ := json.MarshalIndent(args, "", "	")
	fmt.Printf("args JSON string:\n%s\n", b)

	// Output:
	// request:
	// POST /info/henrylee2cn?year=2018&year=2019&t=2019-09-04T18%3A04%3A08%2B08%3A00&pages=1,2,3 HTTP/1.1
	// Host: localhost
	// User-Agent: Go-http-client/1.1
	// Transfer-Encoding: chunked
	// Authorization: Basic 123456
	// Content-Type: application/json;charset=utf-8
	// Cookie: sessionid=987654
	// X-User_id: 123456
	//
	// 83
	// {"AutoBody":"autobody_test","Hobby":["Coding","Mountain climbing"],"email":"henrylee2cn@gmail.com","friendly":true,"pie":3.1415926}
	// 0
	//
	// bind and validate result:
	// error: <nil>
	// args JSON string:
	// {
	// 	"Name": "henrylee2cn",
	// 	"Year": [
	// 		2018,
	// 		2019
	// 	],
	//	"Pages": [
	//		1,
	//		2,
	//		3
	//	],
	// 	"email": "henrylee2cn@gmail.com",
	// 	"friendly": true,
	// 	"pie": 3.1415925,
	// 	"Hobby": [
	// 		"Coding",
	// 		"Mountain climbing"
	// 	],
	// 	"BodyNotFound": null,
	// 	"Authorization": "Basic 123456",
	// 	"SessionID": "987654",
	// 	"AutoBody": "autobody_test",
	// 	"AutoNotFound": null,
	// 	"TimeRFC3339": "2019-09-04T18:04:08+08:00"
	// }
}

func requestExample() *http.Request {
	contentType, bodyReader, _ := httpbody.NewJSONBody(map[string]interface{}{
		"email":    "henrylee2cn@gmail.com",
		"friendly": true,
		"pie":      3.1415926,
		"Hobby":    []string{"Coding", "Mountain climbing"},
		"AutoBody": "autobody_test",
	})
	header := make(http.Header)
	header.Add("Content-Type", contentType)
	header.Add("Authorization", "Basic 123456")
	header.Add("x-user_ID", "123456")
	cookies := []*http.Cookie{
		{Name: "sessionid", Value: "987654"},
	}
	req := newRequest("http://localhost/info/henrylee2cn?year=2018&year=2019&t=2019-09-04T18%3A04%3A08%2B08%3A00&pages=1,2,3", header, cookies, bodyReader)
	req.Method = "POST"
	var w bytes.Buffer
	req.Write(&w)
	fmt.Printf("request:\n%s", strings.Replace(w.String(), "\r\n", "\n", -1))

	bodyReader.(*bytes.Reader).Seek(0, 0)
	return req
}
