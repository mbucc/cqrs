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
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	_ "github.com/mattn/go-sqlite3"
)

type sqlstrings struct {
	CreateSql string
	InsertSql string
	SelectSql string
}

type SqliteEventStore struct {
	dataSourceName string
	tag            string
	eventsInStore  uint64
	db             *sqlx.DB
	sqlcache       map[reflect.Type]sqlstrings
}

func NewSqliteEventStore(dataSourceName string) *SqliteEventStore {
	return &SqliteEventStore{
		dataSourceName: dataSourceName,
		tag:            "db",
		eventsInStore:  0,
		sqlcache:       make(map[reflect.Type]sqlstrings),
	}
}

// SetEventTypes registers event types
// so we can reconsitute into an interface.
// Code in this function is from jmoiron/modl
// and is under MIT License.
func gotypeToSqlite3type(t reflect.Type) string {
	// ToSqlType maps go types to sqlite types.
	switch t.Kind() {
	case reflect.Bool:
		return "integer"
	case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "integer"
	case reflect.Float64, reflect.Float32:
		return "real"
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return "blob"
		}
	}
	switch t.Name() {
	case "NullableInt64":
		return "integer"
	case "NullableFloat64":
		return "real"
	case "NullableBool":
		return "integer"
	case "NullableBytes":
		return "blob"
	case "Time", "NullTime":
		return "datetime"
	}
	return fmt.Sprintf("text")
}

func makeTypedFieldList(fieldnames []string, fieldnameToValue map[string]reflect.Value) string {
	var a []string = make([]string, len(fieldnames))
	for i, name := range fieldnames {
		v := fieldnameToValue[name]
		a[i] = fmt.Sprintf("%s %s", name, gotypeToSqlite3type(v.Type()))
	}
	return strings.Join(a, ", ")
}

func tableName(e Event) string {
	t := reflectx.Deref(reflect.TypeOf(e))
	return t.PkgPath() + "." + t.Name()
}

func (es *SqliteEventStore) eventSql(e Event) sqlstrings {
	m := reflectx.NewMapperFunc(es.tag, strings.ToLower)
	fieldnameToValue := m.FieldMap(reflect.ValueOf(e))

	// Sort fieldnames so our create table SQL
	// matches what Sqlite stores in sqlite_master.
	fieldnames := make([]string, len(fieldnameToValue))
	i := 0
	for name := range fieldnameToValue {
		fieldnames[i] = name
		i += 1
	}
	sort.Strings(fieldnames)

	createfmt := "CREATE TABLE [%s] (%s)"
	insertfmt := "insert into [%s] (%s) values (%s)"
	selectfmt := "select %s from [%s] where aggregate_id = ?"
	tname := tableName(e)
	flist := strings.Join(fieldnames, ", ")
	namedflist := ":" + strings.Join(fieldnames, ", :")
	typedflist := makeTypedFieldList(fieldnames, fieldnameToValue)
	return sqlstrings{
		fmt.Sprintf(createfmt, tname, typedflist),
		fmt.Sprintf(insertfmt, tname, flist, namedflist),
		fmt.Sprintf(selectfmt, flist, tname)}
}

func (es *SqliteEventStore) databaseCreateTableSql(e Event) string {
	var dbsql string
	sqlfmt := "select sql from sqlite_master where type = 'table' and name = '%s'"
	s := fmt.Sprintf(sqlfmt, tableName(e))
	err := es.db.Get(&dbsql, s)
	if err != nil && err != sql.ErrNoRows {
		panic(fmt.Sprintf("cqrs: cannot run '%s': %v", s, err))
	}
	return dbsql
}

func (es *SqliteEventStore) SetEventTypes(events []Event) error {
	var err error

	es.db, err = sqlx.Connect("sqlite3", es.dataSourceName)
	if err != nil {
		panic(fmt.Sprintf("cqrs: can't open sqlite database '%s', %v", es.dataSourceName, err))
	}

	es.sqlcache = make(map[reflect.Type]sqlstrings)

	for _, event := range events {

		es.sqlcache[reflect.TypeOf(event)] = es.eventSql(event)
		newsql := es.sqlcache[reflect.TypeOf(event)].CreateSql

		dbsql := es.databaseCreateTableSql(event)

		if len(dbsql) > 0 {
			// Case matters for Sqlite3!
			// For example, code that works fine
			// when a table has lower-case field names
			// will fail if that same table
			// has camel case field names.
			// So, we use case-sensitive comparison for SQL.
			if newsql != dbsql {
				msgfmt := "Table exists for %T, but SQL differs: '%s' != '%s'"
				panic(fmt.Sprintf(msgfmt, event, dbsql, newsql))
			}
		} else {
			fmt.Printf("cqrs: creating table in %s:'%s'\n", es.dataSourceName, newsql)
			es.db.MustExec(newsql)
		}
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
	if _, err := os.Stat(es.dataSourceName); err == nil {
		if err := os.Remove(es.dataSourceName); err != nil {
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
	var events []Event = make([]Event, es.eventsInStore)
	gobfiles, err := filepath.Glob(fmt.Sprintf("%s/%s-*.gob", es.dataSourceName, aggFilenamePrefix))
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
