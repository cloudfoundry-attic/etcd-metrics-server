package main_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	"github.com/cloudfoundry/yagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Main", func() {
	var nats *natsrunner.NATSRunner
	var etcdRunner *etcdstorerunner.ETCDClusterRunner
	var session *gexec.Session

	BeforeEach(func() {
		nats = natsrunner.NewNATSRunner(4222)
		nats.Start()
		etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001, 1)
		etcdRunner.Start()
	})

	AfterEach(func() {
		nats.Stop()
		etcdRunner.Stop()
		session.Kill().Wait()
	})

	type registration struct {
		Host        string   `json:host`
		Credentials []string `json:credentials`
	}

	It("starts the server correctly", func(done Done) {
		var reg = new(registration)

		receivedAnnounce := make(chan bool)
		nats.MessageBus.Subscribe("vcap.component.announce", func(message *yagnats.Message) {
			err := json.Unmarshal(message.Payload, reg)
			receivedAnnounce <- true
			Ω(err).ShouldNot(HaveOccurred())
		})

		var err error
		serverCmd := exec.Command(metricsServerPath,
			"-jobName", "etcd-diego",
			"-port", "5678",
			"-etcdAddress", "127.0.0.1:5001")
		serverCmd.Env = os.Environ()

		session, err = gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
		Ω(err).ShouldNot(HaveOccurred())

		<-receivedAnnounce

		Eventually(func() error {
			conn, err := net.Dial("tcp", reg.Host)
			if err == nil {
				conn.Close()
			}

			return err
		}, 5).ShouldNot(HaveOccurred())

		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/varz", reg.Host), nil)
		Ω(err).ShouldNot(HaveOccurred())
		req.SetBasicAuth(reg.Credentials[0], reg.Credentials[1])

		resp, err := http.DefaultClient.Do(req)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(resp.Status).Should(ContainSubstring("200"))

		body, err := ioutil.ReadAll(resp.Body)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(body).Should(ContainSubstring("etcd-diego"))
		close(done)
	}, 10)
})
