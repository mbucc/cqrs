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
	"database/sql/driver"
	"testing"
	"github.com/mbucc/cqrs"
	. "github.com/smartystreets/goconvey/convey"
	_ "github.com/mattn/go-sqlite3"
)


type Key string


// For name types, we need to implement Valuer and
// Scanner interfaces, otherwise we will get the error:
//
//  	sql: converting Exec argument #1's type:
//     unsupported type cqrs_test.Key, a string
//
// See https://github.com/jmoiron/sqlx/issues/109


// Value implements the driver.Valuer interface
func (r Key) Value() (driver.Value, error) {
    return string(r), nil
}

// Scan implements the sql.Scanner interface
func (r Key) Scan(src interface{}) error {
    r = Key(string(src.([]uint8)))

    return nil
}

// Event with a user type
type MyEvent struct {
	cqrs.BaseEvent
	MyKey Key
}

// Aggregator that always returns one MyEvent, no matter what the command.
type MyAggregate struct{ id cqrs.AggregateID }

func (eh *MyAggregate) Handle(c cqrs.Command) (events []cqrs.Event, err error) {
	return []cqrs.Event{&MyEvent{
		BaseEvent: cqrs.BaseEvent{Id: c.ID()},
		MyKey:     Key("abc") } }, nil
}

func (eh *MyAggregate) ID() cqrs.AggregateID {
	return cqrs.AggregateID(1)
}

func (eh *MyAggregate) New(id cqrs.AggregateID) cqrs.Aggregator {
	return &MyAggregate{cqrs.AggregateID(1)}
}

func (eh *MyAggregate) ApplyEvents([]cqrs.Event) {}

func TestPersistenceOfUserTypeInEvent(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	Convey("Given an event history", t, func() {

		store := cqrs.NewSqliteEventStore(testdb)
		cqrs.RegisterEventListeners(new(MyEvent), new(NullEventListener))
		cqrs.RegisterCommandAggregator(new(ShoutSomething), &MyAggregate{})
		cqrs.RegisterEventStore(store)


		So(cqrs.SendCommand(&ShoutSomething{1, "hello1"}), ShouldEqual, nil)

		events, err := store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 1)
	})
}