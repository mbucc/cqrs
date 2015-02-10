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
	_ "github.com/mattn/go-sqlite3"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCreateTable(t *testing.T) {

	Convey("Given a new Sqlite3 event store", t, func() {

		store := &SqliteEventStore{DataSourceName: "/tmp/cqrs.db"}
		Convey("Registering an event type creates a table", func() {
			err := store.SetEventTypes([]Event{&HeardEvent{}})
			So(err, ShouldEqual, nil)
			db, err := sql.Open("sqlite3", "/tmp/cqrs.db")
			So(err, ShouldEqual, nil)
			defer db.Close()

			var count int
			row := db.QueryRow("select count(*) from sqlite_master where name = 'cqrs.HeardEvent'")
			err = row.Scan(count)
			So(count, ShouldEqual, 1)
		})
		/*
			Reset(func() {
				store.DeleteAllData()
			})
		*/
	})
}
