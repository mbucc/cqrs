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
	"fmt"
	"github.com/jmoiron/modl"
	_ "github.com/mattn/go-sqlite3"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type SqliteEventStore struct {
	// BUG(mbucc) Modifying the DataSourceName of a SqliteEventStore will break things.
	DataSourceName string
	EventsInStore  uint64
	db             *sql.DB
}


// SetEventTypes registers event types
// so we can reconsitute into an interface.
func (es *SqliteEventStore) SetEventTypes(types []Event) error {
	var err error
	if es.db, err = sql.Open("sqlite3", es.DataSourceName); err != nil {
		return err
	}
	dbmap := &modl.DbMap{Db: es.db, Dialect: modl.SqliteDialect{}}
	for _, event := range types {
		dbmap.AddTable(t)
	}
	if err := dbmap.CreateTablesIfNotExists(); err != nil {
		return err
	}
	return nil
}

func (es *SqliteEventStore) eventToTableName(event Event) string {
	s := fmt.Sprintf("%T", event)
	if strings.HasPrefix(s, "*") {
		s = s[1:]
	}
	return s
}

func (es *SqliteEventStore) DeleteAllData() error {
	if es.db != nil {
		if err := es.db.Close(); err != nil {
			return err
		}
	}
	if _, err := os.Stat(es.DataSourceName); err == nil {
		if err := os.Remove(es.DataSourceName); err != nil {
			return err
		}
	}
	return nil
}

// LoadEventsFor opens the gob file for the aggregator and returns any events found.
// If the file does not exist, an empty list is returned.
func (es *SqliteEventStore) LoadEventsFor(agg Aggregator) ([]Event, error) {
	var events []Event
	return events, nil
}

func (es *SqliteEventStore) GetAllEvents() ([]Event, error) {
	var events []Event = make([]Event, es.EventsInStore)
	gobfiles, err := filepath.Glob(fmt.Sprintf("%s/%s-*.gob", es.DataSourceName, aggFilenamePrefix))
	if err != nil {
		panic(fmt.Sprintf("cqrs: logic error (bad pattern) in GetAllEvents, %v", err))
	}

	for _, fn := range gobfiles {
		newevents, err := filenameToEvents(fn)
		if err != nil {
			return nil, err
		}
		events = append(events, newevents...)
	}

	sort.Sort(BySequenceNumber(events))

	return events, nil

}

// SaveEventsFor persists the events to disk for the given Aggregate.
func (es *SqliteEventStore) SaveEventsFor(agg Aggregator, loaded []Event, result []Event) error {
	return nil
}
