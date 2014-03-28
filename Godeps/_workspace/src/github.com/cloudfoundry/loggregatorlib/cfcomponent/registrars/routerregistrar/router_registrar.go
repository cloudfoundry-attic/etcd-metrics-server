package routerregistrar

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/yagnats"
	uuid "github.com/nu7hatch/gouuid"
	"sync"
	"time"
)

type registrar struct {
	*gosteno.Logger
	mBusClient             yagnats.NATSClient
	routerRegisterInterval time.Duration
	lock                   sync.RWMutex
}

func NewRouterRegistrar(mBusClient yagnats.NATSClient, logger *gosteno.Logger) *registrar {
	return &registrar{mBusClient: mBusClient, Logger: logger}
}

func (r *registrar) RegisterWithRouter(hostname string, port uint32, uris []string) error {
	r.subscribeToRouterStart()
	err := r.greetRouter()
	if err != nil {
		return err
	}
	r.keepRegisteringWithRouter(hostname, port, uris)

	return nil
}

func createInbox() (string, error) {
	uuid, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("_INBOX.%s", uuid), nil
}

func (r *registrar) greetRouter() (err error) {
	response := make(chan []byte, 100)

	callback := func(payload []byte) {
		response <- payload
	}

	inbox, err := createInbox()
	if err != nil {
		return err
	}

	r.mBusClient.Subscribe(inbox, func(msg *yagnats.Message) {
		callback([]byte(msg.Payload))
	})

	r.mBusClient.PublishWithReplyTo(RouterGreetMessageSubject, inbox, []byte{})

	routerRegisterInterval := 20 * time.Second
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	select {
	case msg := <-response:
		routerResponse := &RouterResponse{}
		err = json.Unmarshal(msg, routerResponse)
		if err != nil {
			r.Errorf("Error unmarshalling the greet response: %v\n", err)
		} else {
			routerRegisterInterval = routerResponse.RegisterInterval * time.Second
			r.Infof("Greeted the router. Setting register interval to %v seconds\n", routerResponse.RegisterInterval)
		}
	case <-timer.C:
		err = errors.New("Did not get a response to router.greet!")
	}

	r.lock.Lock()
	r.routerRegisterInterval = routerRegisterInterval
	r.lock.Unlock()

	return err
}

func (r *registrar) subscribeToRouterStart() {
	r.mBusClient.Subscribe(RouterStartMessageSubject, func(msg *yagnats.Message) {
		payload := msg.Payload
		routerResponse := &RouterResponse{}
		err := json.Unmarshal(payload, routerResponse)
		if err != nil {
			r.Errorf("Error unmarshalling the router start message: %v\n", err)
		} else {
			r.Infof("Received router.start. Setting register interval to %v seconds\n", routerResponse.RegisterInterval)
			r.lock.Lock()
			r.routerRegisterInterval = routerResponse.RegisterInterval * time.Second
			r.lock.Unlock()
		}
	})
	r.Info("Subscribed to router.start")

	return
}

func (r *registrar) keepRegisteringWithRouter(hostname string, port uint32, uris []string) {
	go func() {
		timer := time.NewTimer(r.routerRegisterInterval)
		defer timer.Stop()
		for {
			timer.Reset(r.routerRegisterInterval)
			err := r.publishRouterMessage(hostname, port, uris, RouterRegisterMessageSubject)
			if err != nil {
				r.Error(err.Error())
			}
			r.Debug("Reregistered with router")

			<-timer.C
		}
	}()
}

func (r *registrar) UnregisterFromRouter(hostname string, port uint32, uris []string) {
	err := r.publishRouterMessage(hostname, port, uris, RouterUnregisterMessageSubject)
	if err != nil {
		r.Error(err.Error())
	}
	r.Info("Unregistered from router")
}

func (r *registrar) publishRouterMessage(hostname string, port uint32, uris []string, subject string) error {
	message := &RouterMessage{
		Host: hostname,
		Port: port,
		Uris: uris,
	}

	json, err := json.Marshal(message)
	if err != nil {
		return errors.New(fmt.Sprintf("Error marshalling the router message: %v\n", err))
	}

	err = r.mBusClient.Publish(subject, json)
	if err != nil {
		return errors.New(fmt.Sprintf("Publishing %s failed: %v", subject, err))
	}
	return nil
}
