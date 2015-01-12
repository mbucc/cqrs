package main

import (
	"errors"
	"fmt"
	"reflect"
	//  	"log"
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
	commandtype reflect.Type
	handler     CommandHandler
}

// Registers event and command listeners.  Dispatches commands.
type messageDispatcher struct {
	commandHandlers map[reflect.Type]CommandHandler
}

func (md *messageDispatcher) SendCommand(c Command) ([]Event, error) {
	t := reflect.TypeOf(c)
	if h, ok := md.commandHandlers[t]; ok {
		return h.handle(c)
	}
	return nil, errors.New(fmt.Sprint("No handler registered for command ", t))
}

func NewMessageDispatcher(pairs []HandlerPair) (*messageDispatcher, error) {
	m := make(map[reflect.Type]CommandHandler, len(pairs))
	for _, pair := range pairs {
		if _, exists := m[pair.commandtype]; exists {
			return nil, errors.New(fmt.Sprint("More than one handler for ", pair.commandtype))
		}
		m[pair.commandtype] = pair.handler
	}
	return &messageDispatcher{commandHandlers: m}, nil
}

func main() {}
