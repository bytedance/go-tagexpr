# binding [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/bytedance/go-tagexpr/binding)

A powerful HTTP request parameters binder that supports struct tag expression.

## Example

```go
package binding_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bytedance/go-tagexpr/binding"
	"github.com/henrylee2cn/goutil/httpbody"
)

func Example() {
	type InfoRequest struct {
		Name          string   `api:"{path:'name'}"`
		Year          []int    `api:"{query:'year'}"`
		Email         *string  `api:"{body:'email'}{@:email($)}"`
		Friendly      bool     `api:"{body:'friendly'}"`
		Pie           float32  `api:"{body:'pie'}{required:true}"`
		Hobby         []string `api:"{body:'hobby'}"`
		BodyNotFound  *int     `api:"{body:'xxx'}"`
		Authorization string   `api:"{header:'Authorization'}{required:true}{@:$=='Basic 123456'}"`
		SessionID     string   `api:"{cookie:'sessionid'}{required:true}"`
		AutoBody      string
		AutoQuery     string
		AutoNotFound  *string
	}

	args := new(InfoRequest)
	binder := binding.New("api")
	err := binder.BindAndValidate(args, requestExample(), new(testPathParams))

	fmt.Println("bind and validate result:")

	fmt.Printf("error: %v\n", err)

	b, _ := json.MarshalIndent(args, "", "	")
	fmt.Printf("args JSON string:\n%s\n", b)

	// Output:
	// request:
	// POST /info/henrylee2cn?year=2018&year=2019&AutoQuery=autoquery_test HTTP/1.1
	// Host: localhost
	// User-Agent: Go-http-client/1.1
	// Transfer-Encoding: chunked
	// Authorization: Basic 123456
	// Content-Type: application/json;charset=utf-8
	// Cookie: sessionid=987654
	//
	// 83
	// {"AutoBody":"autobody_test","email":"henrylee2cn@gmail.com","friendly":true,"hobby":["Coding","Mountain climbing"],"pie":3.1415926}
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
	// 	"Email": "henrylee2cn@gmail.com",
	// 	"Friendly": true,
	// 	"Pie": 3.1415925,
	// 	"Hobby": [
	// 		"Coding",
	// 		"Mountain climbing"
	// 	],
	// 	"BodyNotFound": null,
	// 	"Authorization": "Basic 123456",
	// 	"SessionID": "987654",
	// 	"AutoBody": "autobody_test",
	// 	"AutoQuery": "autoquery_test",
	// 	"AutoNotFound": null
	// }
}
...
```
