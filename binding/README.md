# binding [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/bytedance/go-tagexpr/binding)

A powerful HTTP request parameters binder that supports struct tag expression.

## Example

```go
package binding_test

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bytedance/go-tagexpr/binding"
	"github.com/henrylee2cn/goutil/httpbody"
)

func Example() {
	type InfoRequest struct {
		Name          string   `api:"{path:'name'}"`
		Year          []int    `api:"{query:'year'}"`
		Email         *string  `api:"{body:'email'}{@:email($)}"`
		Friendly      bool     `api:"{body:'friendly'}"`
		Pie           float32  `api:"{body:'pie'}"`
		Hobby         []string `api:"{body:'hobby'}"`
		Authorization string   `api:"{header:'Authorization'}{required:true}{@:$=='Basic 123456'}"`
		SessionID     string   `api:"{cookie:'sessionid'}{required:true}"`
		AutoBody      string
		AutoQuery     string
		NotFound      *string
	}

	args := new(InfoRequest)
	binder := binding.New("api")
	err := binder.BindAndValidate(args, requestExample(), new(testPathParams))

	fmt.Printf("bind and validate error: %v\n", err)

	b, _ := json.MarshalIndent(args, "", "	")
	fmt.Printf("args JSON string:\n%s\n", b)

	// Output:
	// bind and validate error: <nil>
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
	// 	"Authorization": "Basic 123456",
	// 	"SessionID": "987654",
	// 	"AutoBody": "autobody_test",
	// 	"AutoQuery": "autoquery_test",
	// 	"NotFound": null
	// }
}
```