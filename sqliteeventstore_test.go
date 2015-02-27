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

package cqrs

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	. "github.com/smartystreets/goconvey/convey"
)

func TestCreateTable(t *testing.T) {

	Convey("Given a new Sqlite3 event store", t, func() {

		store := NewSqliteEventStore("/tmp/cqrs.db")
		Convey("Registering an event type creates a table", func() {
			err := store.SetEventTypes([]Event{&HeardEvent{}})
			So(err, ShouldEqual, nil)
			db, err := sql.Open("sqlite3", "/tmp/cqrs.db")
			So(err, ShouldEqual, nil)
			defer db.Close()

			var count int
			row := db.QueryRow("select count(*) from sqlite_master where type = 'table'")
			err = row.Scan(&count)
			So(err, ShouldEqual, nil)
			So(count, ShouldEqual, 1)
		})
		Reset(func() {
			os.Remove("/tmp/cqrs.db")
		})
	})
}

func TestCreateTableSqlMatches(t *testing.T) {

	Convey("Given a new Sqlite3 event store", t, func() {

		store := NewSqliteEventStore("/tmp/cqrs.db")
		Convey("That has an event table", func() {
			err := store.SetEventTypes([]Event{&HeardEvent{}})
			So(err, ShouldEqual, nil)
			db, err := sql.Open("sqlite3", "/tmp/cqrs.db")
			So(err, ShouldEqual, nil)
			defer db.Close()

			var count int
			row := db.QueryRow("select count(*) from sqlite_master where type = 'table'")
			err = row.Scan(&count)
			So(err, ShouldEqual, nil)
			So(count, ShouldEqual, 1)
			Convey("Reopening event store should not panic", func() {
				unregisterAll()
				store = NewSqliteEventStore("/tmp/cqrs.db")
				err := store.SetEventTypes([]Event{&HeardEvent{}})
				So(err, ShouldEqual, nil)
			})
		})
		Reset(func() {
			os.Remove("/tmp/cqrs.db")
		})
	})
}

func TestPersistence(t *testing.T) {

	unregisterAll()

	Convey("Given an event history", t, func() {

		store := NewSqliteEventStore("/tmp/cqrs.db")
		RegisterEventListeners(new(HeardEvent), new(NullEventListener))
		RegisterEventStore(store)
		RegisterCommandAggregator(new(ShoutCommand), NullAggregate{})

		So(SendCommand(&ShoutCommand{1, "hello1"}), ShouldEqual, nil)

		events, err := store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 1)

		So(SendCommand(&ShoutCommand{2, "hello2"}), ShouldEqual, nil)
		So(SendCommand(&ShoutCommand{2, "hello2a"}), ShouldEqual, nil)
		So(SendCommand(&ShoutCommand{3, "hello3"}), ShouldEqual, nil)

		events, err = store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 4)

		Convey("We should be able to reload history", func() {
			unregisterAll()
			RegisterEventListeners(new(HeardEvent), new(NullEventListener))
			RegisterEventStore(store)
			events, err := store.GetAllEvents()
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 4)
			Convey("We should be able to reload history", func() {
				var lastId uint64 = 0
				for _, e := range events {
					So(lastId, ShouldBeLessThan, e.GetSequenceNumber())
					lastId = e.GetSequenceNumber()
				}
			})
		})
		Reset(func() {
			os.Remove("/tmp/cqrs.db")
		})
	})
}

func TestSqliteStorePersistsOldAndNewEvents(t *testing.T) {

	unregisterAll()

	Convey("Given an echo handler and two null listeners", t, func() {

		aggid := AggregateID(1)
		agg := EchoAggregate{aggid}
		store := NewSqliteEventStore("/tmp/cqrs.db")
		RegisterEventListeners(new(HeardEvent), new(NullEventListener))
		RegisterEventStore(store)
		RegisterCommandAggregator(new(ShoutCommand), EchoAggregate{})

		Convey("A ShoutCommand should persist old and new events", func() {
			err := SendCommand(&ShoutCommand{aggid, "hello humanoid1"})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(agg)
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 1)

			err = SendCommand(&ShoutCommand{aggid, "hello humanoid2"})
			So(err, ShouldEqual, nil)
			events, err = store.LoadEventsFor(agg)
			So(len(events), ShouldEqual, 2)
		})
		Reset(func() {
			os.Remove("/tmp/cqrs.db")
		})
	})
}
