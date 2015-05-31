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

// Our toy aggregate only needs the command
// to ensure the business rules are kept,
// so we only have one instance (ID) of this
// aggregate for the entire system.
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
	// A command has one and only one aggregate,
	// so we can return constant here.
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
	// It's possible this event was spawned by some other
	// comand, so don't use contstant here.
	return e.Id
}

//---------------------------------------------------------------------------
//
//                             A G G R E G A T E
//
// Business rules:
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

	store := cqrs.NewSqliteEventStore("/tmp/cqrs.db")
	cqrs.RegisterEventListeners(new(HeardSomething), new(cqrs.NullEventListener))
	cqrs.RegisterEventStore(store)
	cqrs.RegisterCommandAggregator(new(ShoutSomething), EchoAggregate{})

	c := &ShoutSomething{1, "Hello World!"}
	err := cqrs.SendCommand(c)
	if err != nil {
		fmt.Println("cqrs: command %v failed: %v", c, err)
	}
}
