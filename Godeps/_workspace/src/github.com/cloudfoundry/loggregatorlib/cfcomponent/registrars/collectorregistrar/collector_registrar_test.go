package collectorregistrar

import (
	"encoding/json"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent"
	"github.com/cloudfoundry/loggregatorlib/loggertesthelper"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAnnounceComponent(t *testing.T) {
	cfc := getTestCFComponent()
	mbus := fakeyagnats.New()

	called := make(chan *yagnats.Message, 10)
	mbus.Subscribe(AnnounceComponentMessageSubject, func(response *yagnats.Message) {
		called <- response
	})

	registrar := NewCollectorRegistrar(mbus, loggertesthelper.Logger())
	registrar.announceComponent(cfc)

	expectedJson, _ := createYagnatsMessage(t, AnnounceComponentMessageSubject)

	payloadBytes := (<-called).Payload
	assert.Equal(t, expectedJson, payloadBytes)
}

func TestSubscribeToComponentDiscover(t *testing.T) {
	cfc := getTestCFComponent()
	mbus := fakeyagnats.New()

	called := make(chan *yagnats.Message, 10)
	mbus.Subscribe(DiscoverComponentMessageSubject, func(response *yagnats.Message) {
		called <- response
	})

	registrar := NewCollectorRegistrar(mbus, loggertesthelper.Logger())
	registrar.subscribeToComponentDiscover(cfc)

	expectedJson, _ := createYagnatsMessage(t, DiscoverComponentMessageSubject)
	mbus.PublishWithReplyTo(DiscoverComponentMessageSubject, "unused-reply", expectedJson)

	payloadBytes := (<-called).Payload
	assert.Equal(t, expectedJson, payloadBytes)
}

func createYagnatsMessage(t *testing.T, subject string) ([]byte, *yagnats.Message) {

	expected := &AnnounceComponentMessage{
		Type:        "Loggregator Server",
		Index:       0,
		Host:        "1.2.3.4:5678",
		UUID:        "0-abc123",
		Credentials: []string{"user", "pass"},
	}

	expectedJson, err := json.Marshal(expected)
	assert.NoError(t, err)

	yagnatsMsg := &yagnats.Message{
		Subject: subject,
		ReplyTo: "reply_to",
		Payload: expectedJson,
	}

	return expectedJson, yagnatsMsg
}

func getTestCFComponent() cfcomponent.Component {
	return cfcomponent.Component{
		IpAddress:         "1.2.3.4",
		Type:              "Loggregator Server",
		Index:             0,
		StatusPort:        5678,
		StatusCredentials: []string{"user", "pass"},
		UUID:              "abc123",
	}
}
