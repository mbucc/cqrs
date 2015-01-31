// Package cqrs provides a command-query responsibility separation library.
//
// This is currently a work in progress,
// and is not yet used in production.
// You can look at the tests to get an idea
// of what should work properly.
//
// Register your Commands, EventListeners, and EventStore
// and start sending commands.
// The library provides a simple FileSystemEventStorer to persist events.
//
// Commands can have one and only one Aggregate.  One command can generate multiple
// events.  All three (commands, events, aggregates) are tied together by the same
// id: the AggregateID.
package cqrs


import (
	"errors"
	"fmt"
	"reflect"
	"time"
)

var commandAggregator = make(map[reflect.Type]Aggregator)
var eventListeners = make(map[reflect.Type][]EventListener)
var eventStore EventStorer
var registeredEvents []Event

// An AggregateID is a unique identifier for an Aggregator instance.
//
// All Events and Commands are associated with an aggregate instance.
type AggregateID int

// A Command is an action
// that can be accepted or rejected.
// For example, MoreDrinksWench!
// might be a command in a medieval
// misogynistic kind of cafe.
//
// If cqrs encounters a concurrency error
// and your Command implementation supports
// rollbacks, cqrs will try to process the
// command a total of three times before
// it fails.
//
// A concurrency error occurs if two of the
// same command occur at the same time; the
// second command updates the event store
// before the first one does.
type Command interface {
	ID() AggregateID
	SupportsRollback() bool
	BeginTransaction() error
	Commit() error
	Rollback() error
}

// An Aggregator is a concept borrowed
// from domain driven design,
// and defines a "noun" in your system
// that contains enough state to guarantee
// that one (or more) of you business
// rules are kept.
type Aggregator interface {
	CommandHandler
	ID() AggregateID
	ApplyEvents([]Event)
	New(AggregateID) Aggregator
}

// CommandHandler is the interface
// that wraps the Handle(c Command) command.
//
// The implementor will typically:
//   - validate the command data
//   - generate events for a valid command
//   - try to persist events
type CommandHandler interface {
	Handle(c Command) (e []Event, err error)
}

// For tests.
func unregisterAll() {
	commandAggregator = make(map[reflect.Type]Aggregator)
	eventListeners = make(map[reflect.Type][]EventListener)
	eventStore = nil
}

// RegisterCommandAggregator associates a Command with it's Aggregator.
// If RegisterCommandAggregator is called twice with the same Command
// type, it panics.
func RegisterCommandAggregator(c Command, a Aggregator) {
	if a == nil {
		panic("cqrs: can't register a nil Aggregator")
	}
	if c == nil {
		panic("cqrs: can't register an Aggregator to a nil Command")
	}
	atype := reflect.TypeOf(a)

	// The aggregator must be a struct
	// so that we can instantiate it
	// in a way that we get to access
	// it's members in a panic-free way.
	if atype.Kind() != reflect.Struct {
		panic(fmt.Sprintf("cqrs: %v is a %v, not a struct", atype, atype.Kind()))
	}
	t := reflect.TypeOf(c)
	if _, dup := commandAggregator[t]; dup {
		panic(fmt.Sprintf("cqrs: RegisterCommandAggregator called twice for command type %v", t))
	}
	commandAggregator[t] = a.New(AggregateID(0))
}

// RegisterEventListeners associates one or more eventListeners
// with an event type.
//
// It is an error to register an event listener
// after the event store is registered.
// Doing so will cause this method to panic.
func RegisterEventListeners(e Event, a ...EventListener) {
	if e == nil {
		panic("cqrs: can't register a nil Event to eventListeners")
	}
	if eventStore != nil {
		panic("cqrs: cannot register event eventListeners after event store has been registered.")
	}
	t := reflect.TypeOf(e)
	if _, exists := eventListeners[t]; !exists {
		eventListeners[t] = []EventListener{}
	}
	for _, x := range eventListeners {
		if x == nil {
			panic("cqrs: can't register a nil Listener to an event")
		}
	}
	eventListeners[t] = append(eventListeners[t], a...)
	registeredEvents = append(registeredEvents, e)
}

// RegisterEventStore registers which event store to use
// for persisting and loading events.
func RegisterEventStore(es EventStorer) {
	if es == nil {
		panic("cqrs: can't register nil EventStorer.")
	}
	eventStore = es
	eventStore.SetEventTypes(registeredEvents)
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
func publishEvent(e Event) error {
	t := reflect.TypeOf(e)
	if a, ok := eventListeners[t]; ok {
		for _, listener := range a {
			if err := listener.apply(e); err != nil {
				return err
			}
		}
	}
	return nil
}

func processCommand(c Command, agg Aggregator) error {
	if eventStore == nil {
		panic("cqrs: must register the event store before processing commands")
	}
	var oldEvents []Event
	var newEvents []Event
	var err error
	var triesLeft int = 3

	if c.SupportsRollback() {
		c.BeginTransaction()
	} else {
		triesLeft = 1
	}

	for ; triesLeft > 0; triesLeft-- {
		a := agg.New(c.ID())
		oldEvents, err = eventStore.LoadEventsFor(a)
		if err == nil {
			a.ApplyEvents(oldEvents)
			newEvents, err = a.Handle(c)
		}
		if err == nil {
			for _, event := range newEvents {
				if err = publishEvent(event); err != nil {
					break
				}
			}
		}
		if err == nil {
			err = eventStore.SaveEventsFor(a, oldEvents, newEvents)

			// If
			//	- we got a concurrency error
			//	- we have retries left
			// then swallow the error, sleep a little,
			// then try again.
			if err != nil {
				if _, ok := err.(*ErrConcurrency) ; ok {
					if triesLeft > 1 {
						err = nil
						c.Rollback()
						time.Sleep(250 * time.Millisecond)
					}
				}
			}
		}

		// We only retry when we get a concurrency
		// error and have retries left.
		// This case is covered above
		// where the err is set to nil.
		// So if err is not nil here,
		// we have a different error
		// and want to report it to the caller,
		// so stop retrying and exit.

		if err != nil {
			triesLeft = 0
		}
	}
	if c.SupportsRollback() {
		if err != nil {
			c.Rollback()
		} else {
			c.Commit()
		}
	}
	return err
}

// SendCommand instantiates the aggregate associated with this command,
// loads events we've stored for this aggregate,
// processes the command,
// and persists any events generated
// by the command processing.
func SendCommand(c Command) error {
	t := reflect.TypeOf(c)
	if agg, ok := commandAggregator[t]; ok {
		return processCommand(c, agg)
	}
	return errors.New(fmt.Sprint("No handler registered for command ", t))
}

