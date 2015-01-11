package main

import (
	"errors"
	"fmt"
)

// Command handlers are responsible
// for validating commands,
// both as a stand-alone set of data
// as well as in the context
// of the Command's aggregate (I know,
// lots of undefined terms here ...
// see the github wiki).
type CommandHandler interface {
	handle(c Command) (e []Event, err error)
}

// Go maps are not thread-safe.
// Define this type
// so at least building the map
// is thread-safe.
type HandlerPair struct {
	C Command
	H CommandHandler
}

// Registers event and command listeners.  Dispatches commands.
type messageDispatcher struct {
	handlers map[Command]CommandHandler
}

func (md *messageDispatcher) SendCommand(c Command) ([]Event, error) {
	return nil, nil
}

func NewMessageDispatcher(handlers []HandlerPair) (*messageDispatcher, error) {
	m := make(map[Command]CommandHandler, len(handlers))
	for _, p := range handlers {
		if _, exists := m[p.C]; exists {
			return nil, errors.New(fmt.Sprint("MMore than one handler specified for %v", p.C))
		}
		m[p.C] = p.H
	}
	return &messageDispatcher{handlers: m}, nil
}

func main() {}
