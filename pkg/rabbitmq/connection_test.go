package rabbitmq

import (
	"context"
	"crypto/tls"
	"testing"
	"time"

	"github.com/github-tree/sponge/pkg/utils"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var (
	url            = "amqp://guest:guest@192.168.3.37:5672/"
	urlTLS         = "amqps://guest:guest@127.0.0.1:5672/"
	datetimeLayout = "2006-01-02 15:04:05.000"
)

func TestConnectionOptions(t *testing.T) {
	opts := []ConnectionOption{
		WithLogger(nil),
		WithLogger(zap.NewNop()),
		WithReconnectTime(time.Second),
		WithTLSConfig(nil),
		WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		}),
	}

	o := defaultConnectionOptions()
	o.apply(opts...)

}

func TestNewConnection1(t *testing.T) {
	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		defer cancel()
		c, err := NewConnection("")
		assert.Error(t, err)

		c, err = NewConnection(url)
		if err != nil {
			t.Log(err)
			return
		}
		assert.True(t, c.CheckConnected())
		time.Sleep(time.Second)
		c.Close()
	})
}

func TestNewConnection2(t *testing.T) {
	utils.SafeRunWithTimeout(time.Second*2, func(cancel context.CancelFunc) {
		defer cancel()
		// error
		_, err := NewConnection(urlTLS)
		assert.Error(t, err)

		_, err = NewConnection(urlTLS, WithTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		}))
		assert.Error(t, err)
	})
}

func TestConnection_monitor(t *testing.T) {
	c := &Connection{
		url:           urlTLS,
		reconnectTime: time.Second,
		exit:          make(chan struct{}),
		zapLog:        defaultLogger,
		conn:          &amqp.Connection{},
		blockChan:     make(chan amqp.Blocking, 1),
		closeChan:     make(chan *amqp.Error, 1),
		isConnected:   true,
	}

	c.CheckConnected()
	go func() {
		defer func() { recover() }()
		c.monitor()
	}()

	time.Sleep(time.Millisecond * 500)
	c.mutex.Lock()
	c.blockChan <- amqp.Blocking{Active: false}
	c.blockChan <- amqp.Blocking{Active: true, Reason: "the disk is full."}
	c.mutex.Unlock()

	time.Sleep(time.Millisecond * 500)
	c.mutex.Lock()
	c.closeChan <- &amqp.Error{Code: 504, Reason: "connect failed"}
	c.mutex.Unlock()

	time.Sleep(time.Millisecond * 500)
	c.Close()
	time.Sleep(time.Millisecond * 500)
}
