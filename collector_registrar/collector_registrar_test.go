package collector_registrar_test

import (
	"encoding/json"
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/registrars/collectorregistrar"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"

	. "github.com/cloudfoundry-incubator/etcd-metrics/collector_registrar"
)

var _ = Describe("CollectorRegistrar", func() {
	var fakenats *fakeyagnats.FakeYagnats
	var registrar CollectorRegistrar
	var component cfcomponent.Component

	BeforeEach(func() {
		fakenats = fakeyagnats.New()
		registrar = New(fakenats)

		component = cfcomponent.Component{
			IpAddress:         "1.2.3.4",
			Type:              "Some Component",
			Index:             1,
			StatusPort:        5678,
			StatusCredentials: []string{"user", "pass"},
			UUID:              "abc123",
		}
	})

	It("announces the component to the collector", func() {
		err := registrar.RegisterWithCollector(component)
		Ω(err).ShouldNot(HaveOccurred())

		expected := collectorregistrar.NewAnnounceComponentMessage(component)

		expectedJson, err := json.Marshal(expected)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(fakenats.PublishedMessages(collectorregistrar.AnnounceComponentMessageSubject)).Should(ContainElement(
			yagnats.Message{
				Subject: collectorregistrar.AnnounceComponentMessageSubject,
				Payload: expectedJson,
			},
		))
	})

	Context("when a discover request is received", func() {
		It("responds with the component info", func() {
			err := registrar.RegisterWithCollector(component)
			Ω(err).ShouldNot(HaveOccurred())

			expected := collectorregistrar.NewAnnounceComponentMessage(component)

			expectedJson, err := json.Marshal(expected)
			Ω(err).ShouldNot(HaveOccurred())

			fakenats.PublishWithReplyTo(
				collectorregistrar.DiscoverComponentMessageSubject,
				"reply-subject",
				nil,
			)

			Ω(fakenats.PublishedMessages("reply-subject")).Should(ContainElement(
				yagnats.Message{
					Subject: "reply-subject",
					Payload: expectedJson,
				},
			))
		})
	})

	Context("when announcing fails", func() {
		disaster := errors.New("oh no!")

		BeforeEach(func() {
			fakenats.WhenPublishing(collectorregistrar.AnnounceComponentMessageSubject, func() error {
				return disaster
			})
		})

		It("returns the error", func() {
			err := registrar.RegisterWithCollector(component)
			Ω(err).Should(Equal(disaster))
		})
	})

	Context("when subscribing fails", func() {
		disaster := errors.New("oh no!")

		BeforeEach(func() {
			fakenats.WhenSubscribing(collectorregistrar.DiscoverComponentMessageSubject, func() error {
				return disaster
			})
		})

		It("returns the error", func() {
			err := registrar.RegisterWithCollector(component)
			Ω(err).Should(Equal(disaster))
		})
	})
})
