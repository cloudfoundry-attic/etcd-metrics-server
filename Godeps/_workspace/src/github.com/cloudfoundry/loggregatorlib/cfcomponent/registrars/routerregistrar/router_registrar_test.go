package routerregistrar

import (
	"github.com/cloudfoundry/loggregatorlib/loggertesthelper"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestGreetRouter(t *testing.T) {
	routerReceivedChannel := make(chan *yagnats.Message, 10)
	resultChan := make(chan bool)

	mbus := fakeyagnats.New()
	fakeRouter(mbus, routerReceivedChannel)
	registrar := NewRouterRegistrar(mbus, loggertesthelper.Logger())

	go func() {
		err := registrar.greetRouter()
		assert.NoError(t, err)
	}()

	go func() {
		for {
			registrar.lock.RLock()
			if registrar.routerRegisterInterval == 42*time.Second {
				resultChan <- true
				registrar.lock.RUnlock()
				break
			}
			registrar.lock.RUnlock()
			time.Sleep(5 * time.Millisecond)
		}
	}()

	select {
	case <-resultChan:
		assert.Equal(t, len(mbus.Subscriptions("router.greet")), 1)
		assert.Equal(t, len(mbus.Subscriptions("router.register")), 1)
	case <-time.After(2 * time.Second):
		t.Error("Router did not receive a router.start in time!")
	}
}

func TestDefaultIntervalIsSetWhenGreetRouterFails(t *testing.T) {
	routerReceivedChannel := make(chan *yagnats.Message)
	resultChan := make(chan bool)

	mbus := fakeyagnats.New()
	fakeBrokenGreeterRouter(mbus, routerReceivedChannel)
	registrar := NewRouterRegistrar(mbus, loggertesthelper.Logger())

	go func() {
		err := registrar.greetRouter()
		assert.Error(t, err)
	}()

	go func() {
		for {
			registrar.lock.RLock()
			if registrar.routerRegisterInterval == 20*time.Second {
				resultChan <- true
				registrar.lock.RUnlock()
				break
			}
			registrar.lock.RUnlock()
			time.Sleep(5 * time.Millisecond)
		}
	}()

	select {
	case <-resultChan:
	case <-time.After(2 * time.Second):
		t.Error("Default register interval was never set!")
	}
}

func TestDefaultIntervalIsSetWhenGreetWithoutRouter(t *testing.T) {
	resultChan := make(chan bool)

	mbus := fakeyagnats.New()
	registrar := NewRouterRegistrar(mbus, loggertesthelper.Logger())

	go func() {
		err := registrar.greetRouter()
		assert.Error(t, err)
	}()

	go func() {
		for {
			registrar.lock.RLock()
			if registrar.routerRegisterInterval == 20*time.Second {
				resultChan <- true
				registrar.lock.RUnlock()
				break
			}
			registrar.lock.RUnlock()
			time.Sleep(5 * time.Millisecond)
		}
	}()

	select {
	case <-resultChan:
	case <-time.After(32 * time.Second):
		t.Error("Default register interval was never set!")
	}
}

func TestKeepRegisteringWithRouter(t *testing.T) {
	mbus := fakeyagnats.New()
	os.Setenv("LOG_TO_STDOUT", "false")
	routerReceivedChannel := make(chan *yagnats.Message)
	fakeRouter(mbus, routerReceivedChannel)

	registrar := NewRouterRegistrar(mbus, loggertesthelper.Logger())
	registrar.routerRegisterInterval = 50 * time.Millisecond
	registrar.keepRegisteringWithRouter("13.12.14.15", 8083, []string{"foobar.vcap.me"})

	for i := 0; i < 3; i++ {
		time.Sleep(55 * time.Millisecond)
		select {
		case msg := <-routerReceivedChannel:
			assert.Equal(t, `registering:{"host":"13.12.14.15","port":8083,"uris":["foobar.vcap.me"]}`, string(msg.Payload))
		default:
			t.Error("Router did not receive a router.register in time!")
		}
	}
}

func TestSubscribeToRouterStart(t *testing.T) {
	mbus := fakeyagnats.New()
	registrar := NewRouterRegistrar(mbus, loggertesthelper.Logger())
	registrar.subscribeToRouterStart()

	err := mbus.Publish("router.start", []byte(messageFromRouter))
	assert.NoError(t, err)

	resultChan := make(chan bool)
	go func() {
		for {
			registrar.lock.RLock()
			if registrar.routerRegisterInterval == 42*time.Second {
				resultChan <- true
				registrar.lock.RUnlock()
				break
			}
			registrar.lock.RUnlock()
			time.Sleep(5 * time.Millisecond)
		}
	}()

	select {
	case <-resultChan:
	case <-time.After(2 * time.Second):
		t.Error("Router did not receive a router.start in time!")
	}
}

func TestUnregisterFromRouter(t *testing.T) {
	mbus := fakeyagnats.New()
	routerReceivedChannel := make(chan *yagnats.Message, 10)
	fakeRouter(mbus, routerReceivedChannel)

	registrar := NewRouterRegistrar(mbus, loggertesthelper.Logger())
	registrar.UnregisterFromRouter("13.12.14.15", 8083, []string{"foobar.vcap.me"})

	select {
	case msg := <-routerReceivedChannel:
		host := "13.12.14.15"
		assert.Equal(t, `unregistering:{"host":"`+host+`","port":8083,"uris":["foobar.vcap.me"]}`, string(msg.Payload))
	case <-time.After(2 * time.Second):
		t.Error("Router did not receive a router.unregister in time!")
	}
}

const messageFromRouter = `{
  							"id": "some-router-id",
  							"hosts": ["1.2.3.4"],
  							"minimumRegisterIntervalInSeconds": 42
							}`

func fakeRouter(mbus *fakeyagnats.FakeYagnats, returnChannel chan *yagnats.Message) {
	mbus.Subscribe("router.greet", func(msg *yagnats.Message) {
		mbus.Publish(msg.ReplyTo, []byte(messageFromRouter))
	})

	mbus.Subscribe("router.register", func(msg *yagnats.Message) {
		returnChannel <- &yagnats.Message{
			Subject: msg.Subject,
			ReplyTo: msg.ReplyTo,
			Payload: []byte("registering:" + string(msg.Payload)),
		}

		mbus.Publish(msg.ReplyTo, msg.Payload)
	})

	mbus.Subscribe("router.unregister", func(msg *yagnats.Message) {
		returnChannel <- &yagnats.Message{
			Subject: msg.Subject,
			ReplyTo: msg.ReplyTo,
			Payload: []byte("unregistering:" + string(msg.Payload)),
		}
		mbus.Publish(msg.ReplyTo, msg.Payload)
	})
}

func fakeBrokenGreeterRouter(mbus *fakeyagnats.FakeYagnats, returnChannel chan *yagnats.Message) {

	mbus.Subscribe("router.greet", func(msg *yagnats.Message) {
		mbus.Publish(msg.ReplyTo, []byte("garbel garbel"))
	})
}
