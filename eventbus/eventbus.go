package eventbus

import "context"

type Bus interface {
	Publish(topic string, msg any) error
	Subscribe(topic string, handler MessageReceiver)
}

type MessageReceiver interface {
	Receive(ctx context.Context, msg any)
}
