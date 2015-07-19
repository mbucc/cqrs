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
//    STATUS
//
//    Under active development. The API is not stable.
//    Performance degrades exponentially with size of event history.
//
// This implementation is largely a translation of the Edument
// CQRS starter kit (https://github.com/edumentab/cqrs-starter-kit).
// Read and write responsibilities are separated---only a command
// can write data and only a query can read data.
//
// A Command is a request to your system, which is either
// accepted or rejected.  A Command can create zero or more
// events.
//
// Queries subscribe to events, and build a view of your
// event history that is read-only and optimized for speed,
// for example by storing an in-menory, denormalized summary
// that is specific to one screen in your application.
//
// Finally, an Aggregator defines a "consistency-boundary"
// that guarantees a business rule is kept.  An Aggregator
// can only use it's own state and the state in the Command
// to do it's job.
//
//   Notes:
//
//   * only events are persisted.
//
//   * all events are persisted.
//
//   * a Command has one and only one Aggregator
//
//   * Aggregator implementation must be a struct
//
//   * synchronous-only; Commands are processed one-
//     at-a-time, in the order they are received.
//
// From profiling (https://github.com/mbucc/cqrsprof/blob/master/cqrsprof.svg)
// performance degrades exponentially.  Adding snapshots of Aggregators is
// the obvious way to speed things up.
//
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

// An AggregateID is a unique identifier
// for an Aggregator instance.
//
// For example, let's say your application
// is for a poker league, and in every poker game
// the total pay-in must equal the pay-out.
//
// The noun (Aggregator) is PokerGame and
// each game has a different id (AggregateID).
//
type AggregateID uint64

// A Command is an action
// that can be accepted or rejected.
//
// In this implementation,
// every Command is associated
// with one and only one Aggregator.
type Command interface {
	ID() AggregateID
}

// An Aggregator defines a "noun" in your system;
// for example, a PokerGame.
// It's job is to maintain enough state
// so it can ensure
// that one (or more) business
// rules are kept.
// The Aggregator cannot access
// a read-model for state; it can only use
// the state it holds and the Command contents.
//
// Every Command type is registered with one and
// only one Aggregator.
//
// The life-cycle of an Aggregator starts when
// a Command of that type is received
// via the SendCommand routine, which
// creates an instance
// of the Aggregator registered to that Command
// via the Aggregator's New() method.
//
// Next, we get all historical events
// for this Aggregator, and replay them
// through the Aggregator with ApplyEvents.
// Finally, the Aggregator is asked to Handle
// the Command,
// and it validates the Command
// against the "aggregate" state rebuilt by the
// event history replay.
//
// Performance: Add a snapshot event type
// that saves current
// Aggregator state.  When replaying
// the event history, find the most
// recent snapshot event
// and only replay from
// that event forward.
type Aggregator interface {
	CommandHandler
	ID() AggregateID
	ApplyEvents([]Event)
	New(AggregateID) Aggregator
}

// CommandHandler is the interface
// that wraps the Handle(c Command) Command, and will typically:
//   - validate the Command data
//   - generate events for a valid Command
//   - try to persist events
//
// Note that in this implementation,
// the Aggregator interface embeds the CommandHandler.
type CommandHandler interface {
	Handle(c Command) (e []Event, err error)
}

// An Event is something that happened
// as a result of a Command.  It is a fact
// and is written in the past tense.
type Event interface {
	ID() AggregateID

	// When an event is published, it is given a unique sequence number.
	SetSequenceNumber(uint64)
	GetSequenceNumber() uint64

	// It's useful to be able to use an event sequence number
	// as an entity ID.  This method casts the event's sequence
	// number to an AggregateID.
	GetSequenceNumberAsAggregateID() AggregateID
}

// A BaseEvent provides the minimum fields
// and methods to implement an event.
// It is meant to be embedded in your
// custom events.
//
// Note that the annotations are used
// by the SqlEventStore when creating
// fields names for the tables that
// persist the particular event.
// BUG(mbucc) If an event type embeds BaseEvent and shadows Id field, sql won't save the id.
type BaseEvent struct {
	// An event is numbered in the order it was published.
	// Events are published as part of cqrs.SendCommand().
	SequenceNumber uint64 `db:"sequence_number"`
	// The Aggregator instance that processed the Command
	// that generated this event.
	Id AggregateID `db:"aggregate_id"`
}

func (e *BaseEvent) GetSequenceNumber() uint64  { return e.SequenceNumber }
func (e *BaseEvent) SetSequenceNumber(n uint64) { e.SequenceNumber = n }
func (e *BaseEvent) ID() AggregateID            { return e.Id }
func (e *BaseEvent) GetSequenceNumberAsAggregateID() AggregateID  {
	return AggregateID(e.SequenceNumber) 
}

type BySequenceNumber []Event

func (s BySequenceNumber) Len() int      { return len(s) }
func (s BySequenceNumber) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s BySequenceNumber) Less(i, j int) bool {
	return s[i].GetSequenceNumber() < s[j].GetSequenceNumber()
}

// An EventListener is typically a read model,
// for example, an in-memory denormalized summary of your
// data that is very fast to query.
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

// So tests can clear registrations stored at package level.
func UnregisterAll() {
	commandAggregator = make(map[reflect.Type]Aggregator)
	eventListeners = make(map[reflect.Type][]EventListener)
	registeredEvents = nil
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

// BUG(mbucc) Don't allow read model errors stop command processing.
//
//  * Events represent a thing that actually happened,
//    a fact, having an event listener return an error
//    is not the right thing to do.
//
//  * Read models are entirely rebuilt when daemon restarts.
//    So if there is a bug, fix it and restart the daemon.
//
//  * For read models with side effects that could fail (like
//    posting to another web service), they should probably
//    enqueue the event for later processing.
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

// SendCommand instantiates the Aggregator associated with this command,
// loads events we've stored for this Aggregator,
// processes the command,
// publishes and persists any events generated
// by the command processing.
//
// Uses a semaphore to serialize command
// processing.
// Technically, we only need to serialize
// by command type (since every command
// type has one and only one Aggregator, and
// Aggregators are independent).
func SendCommand(c Command) error {
	t := reflect.TypeOf(c)
	if agg, ok := commandAggregator[t]; ok {
		sem <- 1
		err := processCommand(c, agg)
		<-sem
		return err
	}
	return errors.New(fmt.Sprint("No Aggregator registered for command ", t))
}
