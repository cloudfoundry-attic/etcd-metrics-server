package collectorregistrar

import (
	"encoding/json"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/yagnats"
)

type collectorRegistrar struct {
	*gosteno.Logger
	mBusClient yagnats.NATSClient
}

func NewCollectorRegistrar(mBusClient yagnats.NATSClient, logger *gosteno.Logger) *collectorRegistrar {
	return &collectorRegistrar{mBusClient: mBusClient, Logger: logger}
}

func (r collectorRegistrar) RegisterWithCollector(cfc cfcomponent.Component) error {
	r.Debugf("Registering component %s with collect at ip: %s, port: %d, username: %s, password %s", cfc.UUID, cfc.IpAddress, cfc.StatusPort, cfc.StatusCredentials[0], cfc.StatusCredentials[1])
	err := r.announceComponent(cfc)
	r.subscribeToComponentDiscover(cfc)

	return err
}

func (r collectorRegistrar) announceComponent(cfc cfcomponent.Component) error {
	json, err := json.Marshal(NewAnnounceComponentMessage(cfc))
	if err != nil {
		return err
	}
	r.mBusClient.Publish(AnnounceComponentMessageSubject, json)
	return nil
}

func (r collectorRegistrar) subscribeToComponentDiscover(cfc cfcomponent.Component) {
	var callback yagnats.Callback
	callback = func(msg *yagnats.Message) {
		json, err := json.Marshal(NewAnnounceComponentMessage(cfc))
		if err != nil {
			r.Warnf("Failed to marshal response to message [%s]: %s", DiscoverComponentMessageSubject, err.Error())
		}
		r.mBusClient.Publish(msg.ReplyTo, json)
	}

	r.mBusClient.Subscribe(DiscoverComponentMessageSubject, callback)

	return
}
