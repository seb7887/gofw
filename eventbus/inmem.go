package eventbus

import "context"

var _ Bus = (*InMem)(nil)

type InMem struct {
	ch chan any
}

func NewInMemBus() *InMem {
	return &InMem{
		ch: make(chan any, 100),
	}
}

func (b *InMem) Publish(_ string, msg any) error {
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
