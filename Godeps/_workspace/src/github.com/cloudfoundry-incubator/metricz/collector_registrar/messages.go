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
	return &AnnounceComponentMessage{
		Type:        component.Type,
		Index:       component.Index,
		Host:        fmt.Sprintf("%s:%d", component.IpAddress, component.StatusPort),
		UUID:        fmt.Sprintf("%d-%s", component.Index, component.UUID),
		Credentials: component.StatusCredentials,
	}
}
