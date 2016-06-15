package fakes

import (
	"net/http"

	"github.com/cloudfoundry-incubator/cf_http"
	"github.com/cloudfoundry-incubator/etcd-metrics-server/instruments"
)

type Getter struct {
	GetCall struct {
		CallCount int
		Recieves  struct {
			Address string
		}
		Returns struct {
			Error error
		}
	}
}

func (g *Getter) Get(address string) (*http.Response, error) {
	g.GetCall.CallCount++
	g.GetCall.Recieves.Address = address
	if g.GetCall.Returns.Error != nil {
		return nil, g.GetCall.Returns.Error
	}

	client := cf_http.NewClient()
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return instruments.ErrRedirected
	}
	return client.Get(address)
}
