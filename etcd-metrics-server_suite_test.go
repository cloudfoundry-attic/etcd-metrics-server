package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var serverCmdPath string

func TestEtcdMetricsServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Etcd-Metrics-Server Suite")
}
