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
		AutoQuery     string
		AutoNotFound  *string
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
	// POST /info/henrylee2cn?year=2018&year=2019&AutoQuery=autoquery_test HTTP/1.1
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
	// 	"AutoQuery": "autoquery_test",
	// 	"AutoNotFound": null
	// }
}
...
```

## Syntax

The parameter position in HTTP request:

|expression|renameable|description|
|----------|----------|-----------|
|`path:"$name"` or `path:"$name,required"`|Yes|URL path parameter|
|`query:"$name"` or `query:"$name,required"`|Yes|URL query parameter|
|`header:"$name"` or `header:"$name,required"`|Yes|Header parameter|
|`cookie:"$name"` or `cookie:"$name,required"`|Yes|Cookie parameter|
|`form:"$name"` or `form:"$name,required"`|Yes|The field in body, support:<br>`application/x-www-form-urlencoded`,<br>`multipart/form-data`|
|`rawbody:""` or `rawbody:"required"`|Yes|The raw bytes of body|
|`vd:"...(tagexpr validator syntax)"`|Yes|The tagexpr expression of validator|
|`json:"$name"` or `json:"$name,required"`|No|The field in body, support:<br>`application/json`|
|`protobuf:"...(raw syntax)"`|No|The field in body, support:<br>`application/x-protobuf`|

**NOTE:**

- `"$name"` is variable placeholder
- If `"$name"` is empty, use the name of field
- If `"$name"` is `-`, omit the field
- Expression `required` indicates that the parameter is required
- If no position is tagged, binding from body first, followed by URL query
