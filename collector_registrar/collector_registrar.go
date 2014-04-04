package collector_registrar

import (
	"encoding/json"

	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
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
	message := NewAnnounceComponentMessage(component)

	messageJson, err := json.Marshal(message)
	if err != nil {
		return err
	}

	_, err = registrar.natsClient.Subscribe(
		DiscoverComponentMessageSubject,
		func(msg *yagnats.Message) {
			registrar.natsClient.Publish(msg.ReplyTo, messageJson)
		},
	)
	if err != nil {
		return err
	}

	err = registrar.natsClient.Publish(
		AnnounceComponentMessageSubject,
		messageJson,
	)
	if err != nil {
		return err
	}

	return nil
}
