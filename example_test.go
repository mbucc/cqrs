/*
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

package cqrs_test

import (
	"errors"
	"fmt"

	"github.com/mbucc/cqrs"
)

// Our toy aggregate only needs the command state
// to ensure the one business rule is kept, so we
// only have one instance of this aggregate for
// the entire system.
//
// Each aggregate instance must have a unique ID, as
// that's how we know which events in the history are
// be applied to which aggregate.
const HelloWorldAggregateID = 0

//---------------------------------------------------------------------------
//
//                              C O M M A N D S
//
//---------------------------------------------------------------------------

type ShoutSomething struct {
	Id      cqrs.AggregateID
	Comment string
}

func (c *ShoutSomething) ID() cqrs.AggregateID {
	// A command has one and only one aggregate type.
	// The aggregate type associated with this command
	// has only one instance, so we can return the
	// constant ID here.
	return HelloWorldAggregateID
}

//---------------------------------------------------------------------------
//
//                                E V E N T S
//
//---------------------------------------------------------------------------

type HeardSomething struct {
	cqrs.BaseEvent
	Heard string
}

func (e *HeardSomething) ID() cqrs.AggregateID {
	// It's possible this event could be fired by another
	// comand type, so don't use the constant here.
	return e.Id
}

//---------------------------------------------------------------------------
//
//                               Q U E R I E S
//
//---------------------------------------------------------------------------

type EventCount struct {
	N int
}

func (p *EventCount) Apply(e cqrs.Event) error {
	//
	// There is no check for integer overflow here.
	//
	// Actually, handling errors in read models is interesting.
	// The current implementation will stop processing a command
	// if a read model returns an error when Apply'ing an event.
	//
	// However, read models are completely rebuilt from the full
	// event history when the cqrs daemon restarts.  A more robust
	// design would be to log the error, have a read-model mark
	// itself as invalid, deploy a fixed model, and restart the
	// daemon.

	p.N += 1
	return nil
}

// Called when cqrs daemon is restarted.
func (p *EventCount) Reapply(e cqrs.Event) error {
	p.N += 1
	return nil
}

//---------------------------------------------------------------------------
//
//                             A G G R E G A T E
//
// Business rule:
//
//      1. We don't echo an empty string.
//
//---------------------------------------------------------------------------

type EchoAggregate struct{ id cqrs.AggregateID }

func (eh EchoAggregate) Handle(c cqrs.Command) (events []cqrs.Event, err error) {
	events = make([]cqrs.Event, 1)
	c1, ok := c.(*ShoutSomething)
	if !ok {
		return nil, errors.New("invalid command")
	}
	if c1.Comment == "" {
		return nil, errors.New("you must shout something")
	}
	events[0] = &HeardSomething{
		BaseEvent: cqrs.BaseEvent{Id: c1.ID()},
		Heard:     c1.Comment}
	return events, nil
}

func (eh EchoAggregate) ID() cqrs.AggregateID {
	return HelloWorldAggregateID
}

func (eh EchoAggregate) New(id cqrs.AggregateID) cqrs.Aggregator {
	return &EchoAggregate{HelloWorldAggregateID}
}

func (eh EchoAggregate) ApplyEvents([]cqrs.Event) {
	// There is no state this aggregate needs to maintain,
	// so this method is empty.
	//
	// A more interesting aggregate would rebuild whatever
	// non-Command state it needs from it's full event
	// history.  An event is associated with an aggregate
	// if it has that aggregate's ID.
}

func Example() {

	// Since cqrs registers state at the package level,
	// when running tests (which all run within the same
	// process), we clear this package-level state so the
	// tests don't interact.
	cqrs.UnregisterAll()
	ClearTestData()

	store := cqrs.NewSqliteEventStore(testdb)
	count := new(EventCount)

	cqrs.RegisterEventListeners(new(HeardSomething), count)
	cqrs.RegisterEventStore(store)
	cqrs.RegisterCommandAggregator(new(ShoutSomething), EchoAggregate{})

	c := &ShoutSomething{HelloWorldAggregateID, "Hello World!"}
	err := cqrs.SendCommand(c)
	if err != nil {
		fmt.Println("cqrs: command %v failed: %v", c, err)
	}

	c = &ShoutSomething{HelloWorldAggregateID, ""}
	err = cqrs.SendCommand(c)
	if err != nil {
		fmt.Printf("cqrs: command %+v failed: %v\n", c, err)
	}

	fmt.Printf("total events = %v\n", count.N)

	// Output:
	// cqrs: creating table for *cqrs_test.HeardSomething in /tmp/testcqrs.db
	// cqrs: command &{Id:0 Comment:} failed: you must shout something
	// total events = 1
}
