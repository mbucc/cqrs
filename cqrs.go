/*
   Copyright (c) 2013, Edument AB
    Copyright (c) 2015, Mark Bucciarelli <mkbucc@gmail.com>

    Permission to use, copy, modify, and/or distribute this software
    for any purpose with or without fee is hereby granted, provided
    that the above copyright notice and this permission notice
    appear in all copies.

    THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL
    WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED
    WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL
    THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR
    CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
    LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT,
    NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN
    CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

// Package cqrs provides a command-query responsibility separation library.
//
// STATUS: Not used in production yet.  See BUGS section for a few flaws.
//
//
// It was inspired by (and is largely a translation of) the Edument
// CQRS starter kit found at https://github.com/edumentab/cqrs-starter-kit
//
// A command is a request that is made to your system,
// which your system either accepts or rejects.
// An accepted command generates one or more events,
// each of which is published to one or more listeners.
//
// Queries are read-only and are built by event listeners.
// A read model is typically optimized
// (denormalized, etc.) for fast reads.
//
// The responsibility for updating data
// is separated from reading the data.
// Commands update state
// and do not provide any read access to data.
// Event listeners build read models,
// which provide read-only access to data.
//
// Another way that responsibility is spread is
// that business rules are satisfied by
// defining a different Aggregate for each "consistency
// boundary" (the minimum set of data required to
// guarantee that a business rule is kept).
package cqrs

import (
	"errors"
	"fmt"
	"reflect"
	"sync/atomic"
)

const CommandQueueSize = 1000

var commandAggregator = make(map[reflect.Type]Aggregator)
var eventListeners = make(map[reflect.Type][]EventListener)
var eventStore EventStorer
var registeredEvents []Event

var sem chan int = make(chan int, CommandQueueSize)

// First event gets sequence number 1.
var eventSequenceNumber uint64 = 0

// An AggregateID is a unique identifier for an Aggregator instance.
//
// All Events and Commands are associated with an aggregate instance.
//
// This will probably change to use a so-called
// globally unique identifier.
// I started with using an int
// because I didn't understand guids.
// If your random number generator is good,
// a guid should be fine; you would need to
// generate 326,000,000 billion guids before
// the chance of a duplicate hit's 1%.
// There is still the issue of poor database
// index performance,
// but that can be measured
// and I expect it is not a real issue
// for the vast majority of sites
// (and certainly mine!).
//
// Here is a really good writeup on guids:
// http://blog.stephencleary.com/2010/11/few-words-on-guids.html
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
type Command interface {
	ID() AggregateID
}

// An Aggregator is a concept borrowed
// from domain driven design,
// and defines a "noun" in your system
// that contains enough state to guarantee
// that one (or more) of you business
// rules are kept.
//
// This was the hardest concept for me to grasp
// when writing this library; for more information
// on Aggregates, see https://github.com/mbucc/cqrs/wiki/Aggregate
//
// Note that this implementation is a bit simpler
// than others in that the CommandHandler interface
// is embedded in the Aggregator interface.
type Aggregator interface {
	CommandHandler
	ID() AggregateID
	ApplyEvents([]Event)
	New(AggregateID) Aggregator
}

// CommandHandler is the interface
// that wraps the Handle(c Command) command, and will typically:
//   - validate the command data
//   - generate events for a valid command
//   - try to persist events
//
// Note that in this implementation,
// the Aggregator interface embeds the CommandHandler.
type CommandHandler interface {
	Handle(c Command) (e []Event, err error)
}

// An Event is something that happened
// as a result of a command;
// for example, FaceSlapped.
type Event interface {
	ID() AggregateID

	// We count atomically and give each event
	// a unique number.
	// Numbers for events within a command are not
	// guaranteed to be sequential; the only guarantee
	// is that a later event will have a higher number
	// than an earlier event.
	SetSequenceNumber(uint64)
	GetSequenceNumber() uint64
}

// BaseEvent exports the SequenceNumber field
// which must be export for the GetAllEvents() method
// of an EventStorer to work correctly.
//
// If you don't embed BaseEvent in your event,
// make sure the data you need
// to respond to a GetSequenceNumber() call
// is stored as an exported struct field
// so it will can be encoded and decoded
// by the EventStorer.
type BaseEvent struct{ SequenceNumber uint64 }

func (e *BaseEvent) GetSequenceNumber() uint64  { return e.SequenceNumber }
func (e *BaseEvent) SetSequenceNumber(n uint64) { e.SequenceNumber = n }

type BySequenceNumber []Event

func (s BySequenceNumber) Len() int      { return len(s) }
func (s BySequenceNumber) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s BySequenceNumber) Less(i, j int) bool {
	return s[i].GetSequenceNumber() < s[j].GetSequenceNumber()
}

// An EventListener is typically a read model,
// for example, an in-memory denormalized summary of your
// data that is very fast to query.
//
type EventListener interface {
	Apply(e Event) error
	Reapply(e Event) error
}

// An EventStorer is an interface that defines the methods
// that persist events
type EventStorer interface {
	SetEventTypes([]Event) error
	LoadEventsFor(Aggregator) ([]Event, error)
	SaveEventsFor(Aggregator, []Event, []Event) error
	GetAllEvents() ([]Event, error)
}

// For tests.
func unregisterAll() {
	commandAggregator = make(map[reflect.Type]Aggregator)
	eventListeners = make(map[reflect.Type][]EventListener)
	eventStore = nil
	eventSequenceNumber = 0
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
// If RegisterEventListeners is called
// after the event store is registered
// will cause a panic.
//
// BUG(mbucc) RegisterEventListeners should only panic if a new event TYPE is added.
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

func republishEvents() {
	var events []Event
	var err error
	events, err = eventStore.GetAllEvents()
	if err != nil {
		panic(fmt.Sprintf("cqrs: GetAllEvents failed, %v", err))
	}
	for _, e := range events {
		t := reflect.TypeOf(e)
		if a, ok := eventListeners[t]; ok {
			for _, listener := range a {
				if err := listener.Reapply(e); err != nil {
					msg := fmt.Sprintf("cqrs: error reapplying event %v to listener %v", e, a, err)
					panic(msg)
				}
			}
		} else {
			msg := fmt.Sprintf("cqrs: no listener registered for event %v", e)
			panic(msg)
		}
		eventSequenceNumber = e.GetSequenceNumber()
	}
}

// RegisterEventStore defines how events
// are written to and read from
// a persistent store.
// In addition, registering an event store
// triggers a task that re-initializes all read models
// by reading all events from history and republishing
// them to the event listeners.
// Note that it is a fatal error if cqrs
// encounters an error reading the event history
// or reprocessing one of the events in the history.
// Either of the conditions will cause a panic.
//
// Once you have registered an event store,
// you cannot register more event listeners.
// The store needs the full set of event types
// so it can de-serialize structs
// into an array of Event interfaces.
//
// You can only register one event store;
// calling RegisterEventStore a second time will cause a panic.
func RegisterEventStore(es EventStorer) {
	if es == nil {
		panic("cqrs: can't register nil EventStorer.")
	}

	if eventStore != nil {
		panic("cqrs: can't register more than one event store")
	}
	eventStore = es
	eventStore.SetEventTypes(registeredEvents)

	// Re-load read models.
	republishEvents()
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
	e.SetSequenceNumber(atomic.AddUint64(&eventSequenceNumber, 1))
	if a, ok := eventListeners[t]; ok {
		for _, listener := range a {
			if err := listener.Apply(e); err != nil {
				return err
			}
		}
		return nil
	} else {
		return fmt.Errorf("cqrs: no listener registered for event %v", e)
	}
}

func processCommand(c Command, agg Aggregator) error {
	if eventStore == nil {
		panic("cqrs: must register the event store before processing commands")
	}
	var oldEvents []Event
	var newEvents []Event
	var err error

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
	}
	return err
}

// SendCommand instantiates the aggregate associated with this command,
// loads events we've stored for this aggregate,
// processes the command,
// publishes and persists any events generated
// by the command processing.
//
// If the command supports transactions,
// and a concurrency error is encountered,
// SendCommand will try to process the command
// a total of three times.
func SendCommand(c Command) error {
	t := reflect.TypeOf(c)
	if agg, ok := commandAggregator[t]; ok {
		sem <- 1
		err := processCommand(c, agg)
		<-sem
		return err
	}
	return errors.New(fmt.Sprint("No aggregate registered for command ", t))
}
