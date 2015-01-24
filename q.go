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
