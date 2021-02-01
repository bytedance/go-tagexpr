# binding [![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](http://godoc.org/github.com/bytedance/go-tagexpr/v2/binding)

A powerful HTTP request parameters binder that supports struct tag expression.

## Example

```go
func Example() {
	type InfoRequest struct {
		Name          string   `path:"name"`
		Year          []int    `query:"year"`
		Email         *string  `json:"email" vd:"email($)"`
		Friendly      bool     `json:"friendly"`
		Status        string   `json:"status" default:"single"`
		Pie           float32  `json:"pie,required"`
		Hobby         []string `json:",required"`
		BodyNotFound  *int     `json:"BodyNotFound"`
		Authorization string   `header:"Authorization,required" vd:"$=='Basic 123456'"`
		SessionID     string   `cookie:"sessionid,required"`
		AutoBody      string
		AutoNotFound  *string
		TimeRFC3339   time.Time `query:"t"`
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
	// POST /info/henrylee2cn?year=2018&year=2019&t=2019-09-04T18%3A04%3A08%2B08%3A00 HTTP/1.1
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
	// 	"status": "single",
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
...
```

## Syntax

The parameter position in HTTP request:

|expression|renameable|description|
|----------|----------|-----------|
|`path:"$name"` or `path:"$name,required"`|Yes|URL path parameter|
|`query:"$name"` or `query:"$name,required"`|Yes|URL query parameter|
|`raw_body:""` or `raw_body:"required"`|Yes|The raw bytes of body|
|`form:"$name"` or `form:"$name,required"`|Yes|The field in body, support:<br>`application/x-www-form-urlencoded`,<br>`multipart/form-data`|
|`protobuf:"...(raw syntax)"`|No|The field in body, support:<br>`application/x-protobuf`|
|`json:"$name"` or `json:"$name,required"`|No|The field in body, support:<br>`application/json`|
|`header:"$name"` or `header:"$name,required"`|Yes|Header parameter|
|`cookie:"$name"` or `cookie:"$name,required"`|Yes|Cookie parameter|
|`default:"$value"`|Yes|Default parameter|
|`vd:"...(tagexpr validator syntax)"`|Yes|The tagexpr expression of validator|

**NOTE:**

- `"$name"` is variable placeholder
- If `"$name"` is empty, use the name of field
- If `"$name"` is `-`, omit the field
- Expression `required` or `req` indicates that the parameter is required
- `default:"$value"` defines the default value for fallback when no binding is successful
- If no position is tagged, try bind parameters from the body when the request has body,
  <br>otherwise try bind from the URL query
- When there is unexportable and no tags, omit the field
- When there are multiple tags, or exportable and no tags, the order in which to try to bind is:
  1. path
  2. form
  3. query
  4. cookie
  5. header
  6. protobuf
  7. json
  8. default

## Type Unmarshalor

TimeRFC3339-binding function is registered by default.

Register your own binding function for the specified type, e.g.:

```go
MustRegTypeUnmarshal(reflect.TypeOf(time.Time{}), func(v string, emptyAsZero bool) (reflect.Value, error) {
	if v == "" && emptyAsZero {
		return reflect.ValueOf(time.Time{}), nil
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(t), nil
})
```
