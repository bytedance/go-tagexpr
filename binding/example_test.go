package binding_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bytedance/go-tagexpr/binding"
	"github.com/henrylee2cn/goutil/httpbody"
)

func Example() {
	type InfoRequest struct {
		Name          string   `path:"name"`
		Year          []int    `query:"year"`
		Email         *string  `json:"email" vd:"email($)"`
		Friendly      bool     `json:"friendly"`
		Pie           float32  `json:"pie,required"`
		Hobby         []string `json:",required"`
		BodyNotFound  *int     `json:"BodyNotFound"`
		Authorization string   `header:"Authorization,required" vd:"$=='Basic 123456'"`
		SessionID     string   `cookie:"sessionid,required"`
		AutoBody      string
		AutoNotFound  *string
		TimeRFC1123   time.Time `query:"t"`
	}

	args := new(InfoRequest)
	binder := binding.New(nil)
	err := binder.BindAndValidate(args, requestExample(), new(testPathParams))

	fmt.Println("bind and validate result:")

	fmt.Printf("error: %v\n", err)

	b, _ := json.MarshalIndent(args, "", "	")
	fmt.Printf("args JSON string:\n%s\n", b)

	// Output:
	// request:
	// POST /info/henrylee2cn?year=2018&year=2019&t=Sun, 06 Nov 2019 22:49:37 GMT HTTP/1.1
	// Host: localhost
	// User-Agent: Go-http-client/1.1
	// Transfer-Encoding: chunked
	// Authorization: Basic 123456
	// Content-Type: application/json;charset=utf-8
	// Cookie: sessionid=987654
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
	// 	"TimeRFC1123": "2019-11-06T22:49:37Z"
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
	cookies := []*http.Cookie{
		{Name: "sessionid", Value: "987654"},
	}
	req := newRequest("http://localhost/info/henrylee2cn?year=2018&year=2019&t=Sun, 06 Nov 2019 22:49:37 GMT", header, cookies, bodyReader)
	req.Method = "POST"
	var w bytes.Buffer
	req.Write(&w)
	fmt.Printf("request:\n%s", strings.Replace(w.String(), "\r\n", "\n", -1))

	bodyReader.(*bytes.Reader).Seek(0, 0)
	return req
}
