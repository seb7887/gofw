package eventbus

import (
	"context"
	"encoding/json"
	"github.com/nats-io/nats.go"
)

var _ Bus = (*NatsConn)(nil)

type NatsConn struct {
	nc *nats.Conn
}

func NewNatsBus(url string) (*NatsConn, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NatsConn{nc: nc}, nil
}

func (eb *NatsConn) Publish(topic string, msg any) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return eb.nc.Publish(topic, b)
}

func (eb *NatsConn) Subscribe(topic string, handler MessageReceiver) {
	_, err := eb.nc.Subscribe(topic, eb.consumedMessages(context.Background(), handler.Receive))
	if err != nil {
		panic(err)
	}
}

func (eb *NatsConn) consumedMessages(ctx context.Context, receiver func(ctx context.Context, msg any)) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		receiver(ctx, msg.Data)
	}
}
