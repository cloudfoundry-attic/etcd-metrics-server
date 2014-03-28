package auth

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func performMockRequest(args ...string) *httptest.ResponseRecorder {
	handler := func(w http.ResponseWriter, req *http.Request) {
		w.Write([]byte("OK"))
	}
	auth := NewBasicAuth(args[0], []string{"user", "password"})
	wrappedHandler := auth.Wrap(handler)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "http://foo/path", nil)
	if len(args) == 3 {
		req.SetBasicAuth(args[1], args[2])
	}
	wrappedHandler(w, req)
	return w
}

func TestRequiresBasicAuth(t *testing.T) {
	w := performMockRequest("realm")
	assert.Equal(t, w.Code, 401)
	assert.Equal(t, w.HeaderMap.Get("WWW-Authenticate"), `Basic realm="realm"`)
	assert.Equal(t, w.Body.String(), "401 Unauthorized")
}

func TestRequiresPromptsWithGivenRealm(t *testing.T) {
	w := performMockRequest("myrealm")
	assert.Equal(t, w.HeaderMap.Get("WWW-Authenticate"), `Basic realm="myrealm"`)
}

func TestFailsWithBadCredentials(t *testing.T) {
	w := performMockRequest("realm", "baduser", "badpassword")
	assert.Equal(t, w.Code, 401)
}

func TestSucceedsWithGoodCredentials(t *testing.T) {
	w := performMockRequest("realm", "user", "password")
	assert.Equal(t, w.Code, 200)
	assert.Equal(t, w.Body.String(), "OK")
}
