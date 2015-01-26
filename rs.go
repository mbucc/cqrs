package cqrs

import (
	"errors"
	"fmt"
	"reflect"
	//"log"
)

// An aggregate defines a boundary around the
// set of data that must be consistent in order
// to guarantee that a rule is kept.
type Aggregator interface {
	CommandHandler
	ApplyEvents([]Event)
}

// The type of function that runs
// when a command is sent
// to the message dispatcher.
type commandProcessor func(c Command) error

// The message dispatcher
// maps each command type
// to one command processor.
type commandProcessors map[reflect.Type]commandProcessor

// When the dispatcher is instantiated,
// one of the arguments is a map
// that associates one Aggregator
// with each command type.
type Aggregators map[reflect.Type]Aggregator

// An event listener is typically
// a read model.
type EventListeners map[reflect.Type][]EventListener

// Registers event and command listeners.  Dispatches commands.
type messageDispatcher struct {
	handlers  commandProcessors
	listeners EventListeners
}

// Instantiate aggregate associated with this command,
// load all events we've already stored for this aggregate,
// process the command,
// and persist any events that were generated
// as a result of the command processing.
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

// Create anonymous functions for each command
// that process the command (instantiate aggregate,
// load events, process new command,
// store new events from processing).
// Register event listeners.
func NewMessageDispatcher(hr Aggregators, lr EventListeners, es EventStorer) (*messageDispatcher, error) {
	var oldEvents, newEvents []Event
	var err error
	md := new(messageDispatcher)
	m := make(commandProcessors, len(hr))
	for commandtype, agg := range hr {
		m[commandtype] = func(c Command) error {
			a := reflect.New(reflect.TypeOf(agg)).Elem().Interface().(Aggregator)
			if oldEvents, err = es.LoadEventsFor(c.ID()); err != nil {
				return err
			}
			a.ApplyEvents(oldEvents)
			if newEvents, err = a.handle(c); err != nil {
				return err
			}
			for _, event := range newEvents {
				if err = md.PublishEvent(event); err != nil {
					return err
				}
			}
			if err := es.SaveEventsFor(c.ID(), oldEvents, newEvents); err != nil {
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
