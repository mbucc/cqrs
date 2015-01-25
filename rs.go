package main

import (
	"errors"
	"fmt"
	"reflect"
//"log"
)

// In the Edument sample app
// they combine Aggregates
// and command handlers.
type Aggregator interface {
	CommandHandler
	ApplyEvents([]Event)
}

type CommandProcessor func(c Command) error
type CommandProcessors map[reflect.Type]CommandProcessor
type Aggregators map[reflect.Type]Aggregator

type EventListeners map[reflect.Type][]EventListener

// Registers event and command listeners.  Dispatches commands.
type messageDispatcher struct {
	handlers  CommandProcessors
	listeners EventListeners
}

func (md *messageDispatcher) SendCommand(c Command) error {
	t := reflect.TypeOf(c)
	if processor, ok := md.handlers[t]; ok {
		return processor(c)
	}
	return errors.New(fmt.Sprint("No handler registered for command ", t))
}

// Since events represent a thing that actually happened,
// a fact, having an event listener return an error
// is probably not the right thing to do.
// While errors can certainly occur,
// for example, email server or database is down
// or the file system is full,
// a better approach would be to stick the events
// in a durable queue so when the error condition clears
// the listener can successfully do it's thing.
func (md *messageDispatcher) PublishEvent(e Event) error {
	t := reflect.TypeOf(e)
	if a, ok := md.listeners[t]; ok {
		for _, listener := range a {
			if err := listener.apply(e); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewMessageDispatcher(hr Aggregators, lr EventListeners, es EventStorer) (*messageDispatcher, error) {
	md := new(messageDispatcher)
	m := make(CommandProcessors, len(hr))
	for commandtype, agg := range hr {
		m[commandtype] = func(c Command) error {
			a := reflect.New(reflect.TypeOf(agg)).Elem().Interface().(Aggregator)
			a.ApplyEvents(es.LoadEventsFor(c.Id()))
			if events, err := a.handle(c); err == nil {
				for _, event := range events {
					if err = md.PublishEvent(event); err != nil {
						return err
					}
				}
			} else {
				return err
			}
			return nil
		}
	}
	l := make(EventListeners, len(lr))
	for eventtype, listeners := range lr {
		l[eventtype] = listeners
	}
	md.handlers = m
	md.listeners = l
	return md, nil
}

func main() {}
