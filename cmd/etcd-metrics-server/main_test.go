package main_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"

	"github.com/apcera/nats"
	"github.com/cloudfoundry-incubator/etcd-metrics-server/runners"
	"github.com/cloudfoundry/gunk/diegonats"
	"github.com/cloudfoundry/gunk/diegonats/gnatsdrunner"
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/cloudfoundry/storeadapter/storerunner/etcdstorerunner"
	"github.com/gogo/protobuf/proto"
	"github.com/onsi/gomega/gexec"
	"github.com/tedsuo/ifrit"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Etcd Metrics Server", func() {
	const (
		CAFilePath   = "fixtures/etcd-ca.crt"
		CertFilePath = "fixtures/server.crt"
		KeyFilePath  = "fixtures/server.key"
	)

	type registration struct {
		Host        string   `json:host`
		Credentials []string `json:credentials`
	}

	etcdMetricsServerTest := func(sslConfig *etcdstorerunner.SSLConfig, args []string) {
		var gnatsdRunner ifrit.Process
		var natsClient diegonats.NATSClient
		var etcdRunner *etcdstorerunner.ETCDClusterRunner
		var session *gexec.Session

		BeforeEach(func() {
			gnatsdRunner, natsClient = gnatsdrunner.StartGnatsd(4222)
			etcdRunner = etcdstorerunner.NewETCDClusterRunner(5001, 1, sslConfig)
			etcdRunner.Start()
		})

		AfterEach(func() {
			gnatsdRunner.Signal(os.Interrupt)
			Eventually(gnatsdRunner.Wait(), 5).Should(Receive())
			etcdRunner.Stop()
			session.Kill().Wait()
		})

		It("starts the server correctly", func(done Done) {
			var reg = new(registration)

			receivedAnnounce := make(chan bool)
			natsClient.Subscribe("vcap.component.announce", func(message *nats.Msg) {
				err := json.Unmarshal(message.Data, reg)
				receivedAnnounce <- true
				Expect(err).ShouldNot(HaveOccurred())
			})

			var err error
			serverCmd := exec.Command(metricsServerPath, args...)
			serverCmd.Env = os.Environ()

			session, err = gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())

			<-receivedAnnounce

			Eventually(func() error {
				conn, err := net.Dial("tcp", reg.Host)
				if err == nil {
					conn.Close()
				}

				return err
			}, 5).ShouldNot(HaveOccurred())

			req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/varz", reg.Host), nil)
			Expect(err).ShouldNot(HaveOccurred())
			req.SetBasicAuth(reg.Credentials[0], reg.Credentials[1])

			resp, err := http.DefaultClient.Do(req)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(resp.Status).Should(ContainSubstring("200"))

			body, err := ioutil.ReadAll(resp.Body)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(body).Should(ContainSubstring("etcd-diego"))
			Expect(body).Should(ContainSubstring("EtcdIndex"))
			Expect(body).Should(ContainSubstring("RaftIndex"))
			Expect(body).Should(ContainSubstring("RaftTerm"))
			close(done)
		}, 10)

		It("starts the metron notifier correctly", func() {
			var err error
			udpConn, err := net.ListenPacket("udp4", "127.0.0.1:3456")
			Expect(err).ShouldNot(HaveOccurred())

			serverCmd := exec.Command(metricsServerPath, args...)
			serverCmd.Env = os.Environ()

			session, err = gexec.Start(serverCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).ShouldNot(HaveOccurred())

			var nextEvent = func() *events.ValueMetric { return readNextEvent(udpConn) }

			Eventually(nextEvent, 15, 0.1).Should(Equal(&events.ValueMetric{
				Name:  proto.String("IsLeader"),
				Value: proto.Float64(1),
				Unit:  proto.String(runners.MetricUnit),
			}))

		}, 15)
	}

	Context("with tls", func() {
		sslConfig := &etcdstorerunner.SSLConfig{
			CAFile:   CAFilePath,
			CertFile: CertFilePath,
			KeyFile:  KeyFilePath,
		}

		args := []string{
			"-jobName", "etcd-diego",
			"-port", "5678",
			"-etcdScheme", "https",
			"-etcdAddress", "127.0.0.1:5001",
			"-caCert", CAFilePath,
			"-cert", CertFilePath,
			"-key", KeyFilePath,
			"-metronAddress", "127.0.0.1:3456",
			"-reportInterval", "1s",
		}

		etcdMetricsServerTest(sslConfig, args)
	})

	Context("without tls", func() {
		args := []string{
			"-jobName", "etcd-diego",
			"-port", "5678",
			"-etcdAddress", "127.0.0.1:5001",
			"-metronAddress", "127.0.0.1:3456",
			"-reportInterval", "1s",
		}

		etcdMetricsServerTest(nil, args)
	})
})

func readNextEvent(udpConn net.PacketConn) *events.ValueMetric {
	bytes := make([]byte, 1024)
	n, _, err := udpConn.ReadFrom(bytes)
	Expect(err).ShouldNot(HaveOccurred())
	Expect(n).Should(BeNumerically(">", 0))

	receivedBytes := bytes[:n]
	var event events.Envelope
	err = proto.Unmarshal(receivedBytes, &event)
	Expect(err).ShouldNot(HaveOccurred())
	return event.GetValueMetric()
}
