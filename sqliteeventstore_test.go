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
	"database/sql"
	"testing"

	"github.com/mbucc/cqrs"
	. "github.com/smartystreets/goconvey/convey"
	_ "github.com/mattn/go-sqlite3"
)

func TestCreateTable(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	Convey("Given a new Sqlite3 event store", t, func() {

		store := cqrs.NewSqliteEventStore(testdb)
		Convey("cqrs.Registering an event type creates a table", func() {
			err := store.SetEventTypes([]cqrs.Event{&HeardSomething{}})
			So(err, ShouldEqual, nil)
			db, err := sql.Open("sqlite3", testdb)
			So(err, ShouldEqual, nil)
			defer db.Close()

			var count int
			row := db.QueryRow("select count(*) from sqlite_master where type = 'table'")
			err = row.Scan(&count)
			So(err, ShouldEqual, nil)
			So(count, ShouldEqual, 1)
		})
	})
}

func TestCreateTableSqlMatches(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	Convey("Given a new Sqlite3 event store", t, func() {

		store := cqrs.NewSqliteEventStore(testdb)
		Convey("That has an event table", func() {
			err := store.SetEventTypes([]cqrs.Event{&HeardSomething{}})
			So(err, ShouldEqual, nil)
			db, err := sql.Open("sqlite3", testdb)
			So(err, ShouldEqual, nil)
			defer db.Close()

			var count int
			row := db.QueryRow("select count(*) from sqlite_master where type = 'table'")
			err = row.Scan(&count)
			So(err, ShouldEqual, nil)
			So(count, ShouldEqual, 1)
			Convey("Reopening event store should not panic", func() {
				cqrs.UnregisterAll()
				store = cqrs.NewSqliteEventStore(testdb)
				err := store.SetEventTypes([]cqrs.Event{&HeardSomething{}})
				So(err, ShouldEqual, nil)
			})
		})
	})
}

func TestPersistence(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	Convey("Given an event history", t, func() {

		store := cqrs.NewSqliteEventStore(testdb)
		cqrs.RegisterEventListeners(new(HeardSomething), new(NullEventListener))
		cqrs.RegisterEventStore(store)
		cqrs.RegisterCommandAggregator(new(ShoutSomething), &NullAggregate{})

		So(cqrs.SendCommand(&ShoutSomething{1, "hello1"}), ShouldEqual, nil)

		events, err := store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 1)

		So(cqrs.SendCommand(&ShoutSomething{2, "hello2"}), ShouldEqual, nil)
		So(cqrs.SendCommand(&ShoutSomething{2, "hello2a"}), ShouldEqual, nil)
		So(cqrs.SendCommand(&ShoutSomething{3, "hello3"}), ShouldEqual, nil)

		events, err = store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 4)

		Convey("We should be able to reload history", func() {
			cqrs.UnregisterAll()
			cqrs.RegisterEventListeners(new(HeardSomething), new(NullEventListener))
			cqrs.RegisterEventStore(store)
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
	})
}

func TestSqliteStorePersistsOldAndNewEvents(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	Convey("Given an echo handler and two null listeners", t, func() {

		aggid := cqrs.AggregateID(1)
		agg := &EchoAggregate{aggid}
		store := cqrs.NewSqliteEventStore(testdb)
		cqrs.RegisterEventListeners(new(HeardSomething), new(NullEventListener))
		cqrs.RegisterEventStore(store)
		cqrs.RegisterCommandAggregator(new(ShoutSomething), &EchoAggregate{})

		Convey("A ShoutSomething should persist old and new events", func() {
			err := cqrs.SendCommand(&ShoutSomething{aggid, "hello humanoid1"})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(agg)
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 1)

			err = cqrs.SendCommand(&ShoutSomething{aggid, "hello humanoid2"})
			So(err, ShouldEqual, nil)
			events, err = store.LoadEventsFor(agg)
			So(len(events), ShouldEqual, 2)
		})
	})
}
