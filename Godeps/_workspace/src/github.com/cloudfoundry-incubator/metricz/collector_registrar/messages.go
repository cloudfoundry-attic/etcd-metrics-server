package collector_registrar

import (
	"fmt"

	"github.com/cloudfoundry-incubator/metricz"
)

const AnnounceComponentMessageSubject = "vcap.component.announce"
const DiscoverComponentMessageSubject = "vcap.component.discover"

type AnnounceComponentMessage struct {
	Type        string   `json:"type"`
	Index       uint     `json:"index"`
	Host        string   `json:"host"`
	UUID        string   `json:"uuid"`
	Credentials []string `json:"credentials"`
}

func NewAnnounceComponentMessage(component metricz.Component) *AnnounceComponentMessage {
	url := component.URL()
	password, _ := url.User.Password()

	return &AnnounceComponentMessage{
		Type:        component.Name(),
		Index:       component.Index(),
		Host:        url.Host,
		UUID:        fmt.Sprintf("%d-%s", component.Index(), component.UUID()),
		Credentials: []string{url.User.Username(), password}, //component.StatusCredentials,
	}
}
