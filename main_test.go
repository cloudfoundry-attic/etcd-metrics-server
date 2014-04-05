package main_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	"github.com/cloudfoundry/yagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vito/cmdtest"
)

var _ = Describe("Main", func() {
	var nats *natsrunner.NATSRunner
	var etcdRunner *etcdstorerunner.ETCDClusterRunner

	BeforeEach(func() {
		nats = natsrunner.NewNATSRunner(4222)
		nats.Start()
		etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001, 1)
		etcdRunner.Start()
	})

	AfterEach(func() {
		nats.Stop()
		etcdRunner.Stop()
	})

	type registration struct {
		Host        string   `json:host`
		Credentials []string `json:credentials`
	}

	It("starts the server correctly", func() {
		var reg = new(registration)
		nats.MessageBus.Subscribe("vcap.component.announce", func(message *yagnats.Message) {
			err := json.Unmarshal(message.Payload, reg)
			Ω(err).ShouldNot(HaveOccurred())
		})

		metricsServerPath, err := cmdtest.Build("github.com/cloudfoundry-incubator/etcd-metrics-server")
		Ω(err).ShouldNot(HaveOccurred())
		serverCmd := exec.Command(metricsServerPath,
			"-port", "5678",
			"-etcdAddress", "127.0.0.1:5001")
		serverCmd.Env = os.Environ()
		session, err := cmdtest.Start(serverCmd)
		defer func() {
			session.Cmd.Process.Kill()
		}()
		Ω(err).ShouldNot(HaveOccurred())

		Eventually(func() (err error) {
			_, err = net.Dial("tcp", reg.Host)
			return err
		}, 15, 0.1).ShouldNot(HaveOccurred())

		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/varz", reg.Host), nil)
		Ω(err).ShouldNot(HaveOccurred())
		req.SetBasicAuth(reg.Credentials[0], reg.Credentials[1])

		resp, err := http.DefaultClient.Do(req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(resp.Status).Should(ContainSubstring("200"))
	})
})
