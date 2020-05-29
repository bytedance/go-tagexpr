package binding

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBody(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("abc")
	req := &http.Request{
		Body: ioutil.NopCloser(&buf),
	}
	body, err := GetBody(req)
	assert.NoError(t, err)
	b, err := ioutil.ReadAll(body)
	assert.NoError(t, err)
	assert.Equal(t, []byte("abc"), b)
	body.Reset()
	assert.Equal(t, []byte("abc"), body.bodyBytes)
	b, err = ioutil.ReadAll(body)
	assert.NoError(t, err)
	assert.Equal(t, []byte("abc"), b)
}
