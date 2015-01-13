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

// Maps a command to its handler.
type HandlerRegistry map[reflect.Type]CommandHandler

// Registers event and command listeners.  Dispatches commands.
type messageDispatcher struct {
	registry HandlerRegistry
}

func (md *messageDispatcher) SendCommand(c Command) ([]Event, error) {
	t := reflect.TypeOf(c)
	if h, ok := md.registry[t]; ok {
		return h.handle(c)
	}
	return nil, errors.New(fmt.Sprint("No handler registered for command ", t))
}

func NewMessageDispatcher(hr HandlerRegistry) (*messageDispatcher, error) {
	m := make(HandlerRegistry, len(hr))
	for commandtype, handler := range hr {
		m[commandtype] = handler
	}
	return &messageDispatcher{registry: m}, nil
}

func main() {}
