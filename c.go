// Package cqrs provides a command-query responsibility separation library.
// 
// You define Commands, Aggregates, Events, and EventListeners and then MessageDispatch
// away.  The library provides a simple FileSystemEventStorer to persist events.
// 
// Commands can have one and only one Aggregate.  One command can generate multiple
// events.  All three (commands, events, aggregates) are tied together by the same
// id: the AggregateID.
// 
// For simplicity, the AggregateID type is a wrapper around an int instead of the more
// typical GUID. So, every client will need to get a pool of id's from the backend.
// 
// It is not suitable for large volume applications, as the conflict resolution is
// primitive, simply returning an error instead of retrying.
package cqrs

// An AggregateID is a unique identifier for an Aggregator instance.
//
// All Events and Commands are associated with an aggregate instance.
type AggregateID int

// A Command is an action
// that can be accepted or rejected.
// For example, MoreDrinksWench!
// might be a command in a medieval
// misogynistic kind of cafe.
type Command interface {
	ID() AggregateID
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
