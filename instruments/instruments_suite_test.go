package instruments_test

import (
	"github.com/cloudfoundry-incubator/cf_http"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
	"time"
)

var timeout time.Duration

func TestInstruments(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Instruments Suite")
}

var _ = BeforeSuite(func() {
	timeout = 1 * time.Second
	cf_http.Initialize(timeout)
})
