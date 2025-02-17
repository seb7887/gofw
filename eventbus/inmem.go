package eventbus

import "context"

type InMem struct {
	ch chan Message
}

func NewInMemBus() *InMem {
	return &InMem{
		ch: make(chan Message, 100),
	}
}

func (b *InMem) Publish(_ string, msg Message) error {
	b.ch <- msg
	return nil
}

func (b *InMem) Subscribe(_ string, handler MessageReceiver) {
	go func() {
		for m := range b.ch {
			handler.Receive(context.Background(), m)
		}
	}()
}
