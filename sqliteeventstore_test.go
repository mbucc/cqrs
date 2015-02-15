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
