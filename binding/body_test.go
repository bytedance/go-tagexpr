package binding

import (
	"bytes"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBody(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("abc")
	body := newBody(&buf)
	b, err := ioutil.ReadAll(body)
	assert.NoError(t, err)
	assert.Equal(t, []byte("abc"), b)
	body.renew()
	assert.Equal(t, []byte("abc"), body.bodyBytes)
	b, err = ioutil.ReadAll(body)
	assert.NoError(t, err)
	assert.Equal(t, []byte("abc"), b)
}
