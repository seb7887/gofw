# eventbus

The **eventbus** package provides a flexible abstraction for communication between components through message publishing and subscription. With generic interfaces, it enables integration of various message bus implementations, such as an in-memory bus and a NATS-based bus.

## Features

- **Generic Abstraction:** Defines interfaces for publishing and subscribing to messages.
- **Multiple Implementations:**
    - **InMem:** An in-memory bus implemented using channels.
    - **NATS:** A bus based on the popular NATS distributed messaging system.
- **Clear Interfaces:**
    - `Bus`: Interface for publishing and subscribing to messages.
    - `Message`: Defines the structure and required methods for messages.
    - `MessageReceiver`: Defines the method that message receivers must implement.

## Installation

```bash
go get github.com/yourusername/eventbus
```

## Usage
Defining Messages and Receivers
Messages must implement the Message interface:

```go
type Message interface {
    Headers() map[string]string
    Payload() []byte
    Serialize() []byte
}
```

Message receivers must implement the MessageReceiver interface:

```go

type MessageReceiver interface {
    Receive(ctx context.Context, msg Message)
}
``` 

### Using the In-Memory Bus (InMem)
The in-memory implementation uses a buffered channel to manage messages.

Create an in-memory bus:

```go
bus := NewInMemBus()
```
#### Subscribe to a topic:

```go
bus.Subscribe("topic", yourHandler)
```

#### Publish a message:

```go
bus.Publish("topic", yourMessage)
```

Note: In the InMem implementation, the topic parameter is ignored.

### Using the NATS-Based Bus
To use the NATS-based implementation, ensure that a NATS server is running and accessible.

Create a NATS-based bus:

```go

bus, err := NewNatsBus[YourMessageType]("nats://localhost:4222")
if err != nil {
// Handle the error appropriately
}
```

#### Subscribe to a topic:

```go
bus.Subscribe("topic", yourHandler)
```

#### Publish a message:

```go
bus.Publish("topic", yourMessage)
```

Note: The generic parameter YourMessageType must implement the Message interface.

### Complete Example
Below is a basic example using the in-memory implementation:

```go
package main

import (
"context"
"fmt"
"github.com/yourusername/eventbus"
)

// MyMessage implements the Message interface
type MyMessage struct {
content string
}

func (m *MyMessage) Headers() map[string]string {
return map[string]string{"type": "example"}
}

func (m *MyMessage) Payload() []byte {
return []byte(m.content)
}

func (m *MyMessage) Serialize() []byte {
return []byte(m.content)
}

func (m *MyMessage) Deserialize() error {
// Implement deserialization if necessary
return nil
}

// MyReceiver implements the MessageReceiver interface
type MyReceiver struct{}

func (r *MyReceiver) Receive(ctx context.Context, msg eventbus.Message) {
fmt.Println("Message received:", string(msg.Payload()))
}

func main() {
// Create the in-memory bus
bus := eventbus.NewInMemBus()

    // Create a receiver and subscribe to a topic
    receiver := &MyReceiver{}
    bus.Subscribe("example", receiver)

    // Create and publish a message
    msg := &MyMessage{content: "Hello, world!"}
    bus.Publish("example", msg)

    // Prevent the program from exiting immediately (for demonstration purposes)
    select {}
}
```
