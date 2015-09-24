package collector_registrar_test

import (
	"encoding/json"
	"errors"

	"github.com/apcera/nats"
	"github.com/cloudfoundry/gunk/diegonats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal-golang/lager"

	"github.com/cloudfoundry-incubator/metricz"
	. "github.com/cloudfoundry-incubator/metricz/collector_registrar"
)

var _ = Describe("CollectorRegistrar", func() {
	var fakenats *diegonats.FakeNATSClient
	var registrar CollectorRegistrar
	var component metricz.Component

	BeforeEach(func() {
		fakenats = diegonats.NewFakeClient()
		registrar = New(fakenats)

		var err error
		component, err = metricz.NewComponent(
			lager.NewLogger("test-component"),
			"Some Component",
			1,
			nil,
			5678,
			[]string{"user", "pass"},
			nil,
		)
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("announces the component to the collector", func() {
		err := registrar.RegisterWithCollector(component)
		Ω(err).ShouldNot(HaveOccurred())

		expected := NewAnnounceComponentMessage(component)

		expectedJson, err := json.Marshal(expected)
		Ω(err).ShouldNot(HaveOccurred())

		Ω(fakenats.PublishedMessages(AnnounceComponentMessageSubject)).Should(ContainElement(
			&nats.Msg{
				Subject: AnnounceComponentMessageSubject,
				Data:    expectedJson,
			},
		))
	})

	Context("when a discover request is received", func() {
		It("responds with the component info", func() {
			err := registrar.RegisterWithCollector(component)
			Ω(err).ShouldNot(HaveOccurred())

			expected := NewAnnounceComponentMessage(component)

			expectedJson, err := json.Marshal(expected)
			Ω(err).ShouldNot(HaveOccurred())

			fakenats.PublishRequest(
				DiscoverComponentMessageSubject,
				"reply-subject",
				nil,
			)

			Ω(fakenats.PublishedMessages("reply-subject")).Should(ContainElement(
				&nats.Msg{
					Subject: "reply-subject",
					Data:    expectedJson,
				},
			))
		})
	})

	Context("when announcing fails", func() {
		disaster := errors.New("oh no!")

		BeforeEach(func() {
			fakenats.WhenPublishing(AnnounceComponentMessageSubject, func(*nats.Msg) error {
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
			fakenats.WhenSubscribing(DiscoverComponentMessageSubject, func(nats.MsgHandler) error {
				return disaster
			})
		})

		It("returns the error", func() {
			err := registrar.RegisterWithCollector(component)
			Ω(err).Should(Equal(disaster))
		})
	})
})
