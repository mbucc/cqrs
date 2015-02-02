package cqrs

import (
	"fmt"
)

// A concurrency error occurs if,
// after an Aggregator has loaded old events from the event store
// and before it has persisted new events resulting from the command processing,
// another command of the same type comes in
// and completes it's processing.
//
// The check for a consistency error is simple: when writing new events to the store,
// we check that the number of events on file
// are the same as the number of events loaded
// when the command processing began.
type ErrConcurrency struct {
	eventCountNow   int
	eventCountStart int
	aggregate       Aggregator
	newEvents       []Event
}

func (e *ErrConcurrency) Error() string {
	return fmt.Sprintf("cqrs: concurrency violation for aggregate %v, %d (start) != %d (now)",
		e.aggregate, e.eventCountStart, e.eventCountNow)
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
type BaseEvent struct {
	SequenceNumber uint64
}

func (e *BaseEvent) GetSequenceNumber() uint64 {
	return e.SequenceNumber
}

func (e *BaseEvent) SetSequenceNumber(n uint64) {
	e.SequenceNumber = n
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
//
type EventListener interface {
	apply(e Event) error
	reapply(e Event) error
}

// An EventStorer is an interface that defines the methods
// that persist events
type EventStorer interface {
	SetEventTypes([]Event)
	LoadEventsFor(Aggregator) ([]Event, error)
	SaveEventsFor(Aggregator, []Event, []Event) error
	GetAllEvents() ([]Event, error)
}

