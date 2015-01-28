// Package cqrs provides a command-query responsibility separation library.
//
// This is currently a work in progress,
// see the Bugs section of http://godoc.org/github.com/mbucc/cqrs#pkg-note-bug
// for what needs to be finished.
//
// Register your Commands, EventListeners, and EventStore
// and start sending commands.
// The library provides a simple FileSystemEventStorer to persist events.
//
// Commands can have one and only one Aggregate.  One command can generate multiple
// events.  All three (commands, events, aggregates) are tied together by the same
// id: the AggregateID.
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
	SupportsRollback() bool
	BeginTransaction() error
	Commit() error
	Rollback() error
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
