package auth_test

import (
	. "github.com/cloudfoundry-incubator/metricz/auth"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"net/http/httptest"
)

var _ = Describe("Testing with Ginkgo", func() {
	It("requires basic auth", func() {
		w := performMockRequest("realm")
		Ω(w.Code).Should(Equal(401))
		Ω(w.HeaderMap.Get("WWW-Authenticate")).Should(Equal(`Basic realm="realm"`))
		Ω(w.Body.String()).Should(Equal("401 Unauthorized"))
	})

	It("requires prompts with given realm", func() {
		w := performMockRequest("myrealm")
		Ω(w.HeaderMap.Get("WWW-Authenticate")).Should(Equal(`Basic realm="myrealm"`))
	})

	It("fails with bad credentials", func() {
		w := performMockRequest("realm", "baduser", "badpassword")
		Ω(w.Code).Should(Equal(401))
	})

	It("succeeds with good credentials", func() {
		w := performMockRequest("realm", "user", "password")
		Ω(w.Code).Should(Equal(200))
		Ω(w.Body.String()).Should(Equal("OK"))
	})
})

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
