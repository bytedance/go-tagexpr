package binding

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBody(t *testing.T) {
	const USE_GZIP = true
	var buf bytes.Buffer
	if USE_GZIP {
		w := gzip.NewWriter(&buf)
		_, err := w.Write([]byte("abc"))
		assert.NoError(t, err)
		err = w.Flush()
		assert.NoError(t, err)
	} else {
		buf.WriteString("abc")
	}
	req := &http.Request{
		Body: ioutil.NopCloser(&buf),
	}
	if USE_GZIP {
		req.Header = map[string][]string{
			"Content-Encoding": []string{"gzip"},
		}
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
