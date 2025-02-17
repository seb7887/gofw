package eventbus

import "context"

type Bus[T Message] interface {
	Publish(topic string, msg Message) error
	Subscribe(topic string, handler MessageReceiver)
}

type Message interface {
	Headers() map[string]string
	Payload() []byte
	Serialize() []byte
}

type MessageReceiver interface {
	Receive(ctx context.Context, msg Message)
}
