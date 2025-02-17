package eventbus

import (
	"context"
	"encoding/json"
	"github.com/nats-io/nats.go"
)

type NatsConn[T Message] struct {
	nc *nats.Conn
}

func NewNatsBus[T Message](url string) (*NatsConn[T], error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, err
	}
	return &NatsConn[T]{nc: nc}, nil
}

func (eb *NatsConn[T]) Publish(topic string, msg Message) error {
	return eb.nc.Publish(topic, msg.Serialize())
}

func (eb *NatsConn[T]) Subscribe(topic string, handler MessageReceiver) {
	_, err := eb.nc.Subscribe(topic, eb.consumedMessages(context.Background(), handler.Receive))
	if err != nil {
		panic(err)
	}
}

func (eb *NatsConn[T]) consumedMessages(ctx context.Context, receiver func(ctx context.Context, msg Message)) func(*nats.Msg) {
	return func(msg *nats.Msg) {
		receiver(ctx, deserialize[T](msg))
	}
}

func deserialize[T any](message *nats.Msg) T {
	var msg T
	_ = json.Unmarshal(message.Data, &msg)
	return msg
}
