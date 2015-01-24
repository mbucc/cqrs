package main

// An event is something that happened
// as a result of a command;
// for example, FaceSlapped.
type Event interface {
	Id() AggregateId
}

type EventListener interface {
	apply(e Event) error
}

// Persist and restore events.
type EventStore interface {
	LoadEventsFor(AggregateId) []Event
	SaveEventsFor(AggregateId, []Event, []Event)
}

type NullEventStore struct {}

func (es *NullEventStore) LoadEventsFor(id AggregateId) []Event {
	return []Event{}
}

func (es *NullEventStore) SaveEventsFor(id AggregateId, loaded []Event, result []Event) {
}
