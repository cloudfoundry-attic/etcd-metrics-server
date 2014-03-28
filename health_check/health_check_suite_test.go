package health_check_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestHealth_check(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Health_check Suite")
}
