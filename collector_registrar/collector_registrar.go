package collector_registrar

import (
	"encoding/json"

	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/registrars/collectorregistrar"
	"github.com/cloudfoundry/yagnats"
)

type CollectorRegistrar interface {
	RegisterWithCollector(cfcomponent.Component) error
}

type natsCollectorRegistrar struct {
	natsClient yagnats.NATSClient
}

func New(natsClient yagnats.NATSClient) CollectorRegistrar {
	return &natsCollectorRegistrar{
		natsClient: natsClient,
	}
}

func (registrar *natsCollectorRegistrar) RegisterWithCollector(component cfcomponent.Component) error {
	message := collectorregistrar.NewAnnounceComponentMessage(component)

	messageJson, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = registrar.natsClient.Subscribe(
		collectorregistrar.DiscoverComponentMessageSubject,
		func(msg *yagnats.Message) {
			registrar.natsClient.Publish(msg.ReplyTo, messageJson)
		},
	)
	if err != nil {
		return err
	}

	err = registrar.natsClient.Publish(
		collectorregistrar.AnnounceComponentMessageSubject,
		messageJson,
	)
	if err != nil {
		return err
	}

	return nil
}
