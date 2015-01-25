/*
Package cqrs provides a simple command-query responsibility separation library.

You define Commands, Aggregates, Events, and EventListeners and then MessageDispatch
away.  The library provides a simple FileSystemEventStorer to persist events.

Commands can have one and only one Aggregate.  One command can generate multiple
events.  All three (commands, events, aggregates) are tied together by the same
id: the AggregateId.

For simplicity, the AggregateId type is a wrapper around an int instead of the more
typical GUID. So, every client will need to get a pool of id's from the backend.

It is not suitable for large volume applications, as the conflict resolution is
primitive, simply returning an error instead of retrying.
*/

package cqrs

// Each aggregate instance is identified by a unique id.
// All Events and Commands are associated with an aggregate instance.
type AggregateId int

// A command is an action
// that can be accepted or rejected.
// For example, MoreDrinksWench!
// might be a command in a medieval
// misogynistic kind of cafe.
type Command interface {
	Id() AggregateId
}


// Command handlers are responsible
// for validating commands,
// both as a stand-alone set of data
// as well as in the context
// of the Command's aggregate.
type CommandHandler interface {
        handle(c Command) (e []Event, err error)
}
