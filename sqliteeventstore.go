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
	"reflect"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/reflectx"
	_ "github.com/mattn/go-sqlite3"
)

const (
	// The field name used for AggregateID in event structs.
	// BUG(mbucc) Dry up handling of AggregateID field name (here and in BaseEvent).
	AggregateIdFieldName = "aggregate_id"

	// The CREATE TABLE clause must be upper case,
	// as the check that the table schema is correct
	// is case-sensitive, and Sqlite upper cases the
	// CREATE TABLE clause when it stores the schema
	// BUG(mbucc) Ensure no SQL-injection via wierd chars in Go struct names and fields.
	CreateFmt                 = "CREATE TABLE [%s] (%s)"
	InsertFmt                 = "insert into [%s] (%s) values (%s)"
	SelectByAggregateIdFmt    = "select %s from [%s] where " + AggregateIdFieldName + " = ?"
	SelectAllFmt              = "select %s from [%s]"
	CountFmt                  = "select count(*) from [%s] where " + AggregateIdFieldName + " = ?"
	CountAllFmt               = "select count(*) from [%s]"
	CreateIndexAggregateIdFmt = "create index [%s.aggregate_id] on [%s] (" + AggregateIdFieldName + ")"

	// The tag label used in event structs to define field names.
	DbTag = "db"
)

type sqlstrings struct {
	Count                  string
	CountAll               string
	Create                 string
	Insert                 string
	Select                 string
	SelectAll              string
	TableName              string
	CreateIndexAggregateId string
}

type eventInfo struct {
	queries sqlstrings
}

// A SqliteEventStore persists events to a Sqlite3 database.
type SqliteEventStore struct {
	datasource string
	db         *sqlx.DB
	eventinfo  map[reflect.Type]eventInfo
}

func NewSqliteEventStore(datasource string) *SqliteEventStore {
	UnregisterAll()
	return &SqliteEventStore{
		datasource: datasource,
		eventinfo:  make(map[reflect.Type]eventInfo),
	}
}

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

func (es *SqliteEventStore) loadEventInfo(e Event) {

	m := reflectx.NewMapperFunc(DbTag, strings.ToLower)
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

	tname := tableName(e)
	flist := strings.Join(fieldnames, ", ")
	namedflist := ":" + strings.Join(fieldnames, ", :")
	typedflist := makeTypedFieldList(fieldnames, fieldnameToValue)

	tmp := es.eventinfo[reflect.TypeOf(e)]
	tmp.queries = sqlstrings{
		fmt.Sprintf(CountFmt, tname),
		fmt.Sprintf(CountAllFmt, tname),
		fmt.Sprintf(CreateFmt, tname, typedflist),
		fmt.Sprintf(InsertFmt, tname, flist, namedflist),
		fmt.Sprintf(SelectByAggregateIdFmt, flist, tname),
		fmt.Sprintf(SelectAllFmt, flist, tname),
		tableName(e),
		fmt.Sprintf(CreateIndexAggregateIdFmt, tname, tname)}
	es.eventinfo[reflect.TypeOf(e)] = tmp
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

// SetEventTypes registers event types
// so we can reconsitute into an interface.
// SetEventTypes opens a connection to the database,
// and creates tables to store events if necessary.
//
// If it can't open the database, it panics.
//
// If the table already exists, but has a different
// structure than required by the event, it panics.
//
// Note that you can control the column names by using
// a "db" tag in your event struct; see BaseEvent for
// examples.
func (es *SqliteEventStore) SetEventTypes(events []Event) error {
	var err error

	es.db, err = sqlx.Connect("sqlite3", es.datasource)
	if err != nil {
		panic(fmt.Sprintf("cqrs: can't open sqlite database '%s', %v", es.datasource, err))
	}

	es.eventinfo = make(map[reflect.Type]eventInfo)

	for _, event := range events {

		es.loadEventInfo(event)
		q := es.eventinfo[reflect.TypeOf(event)].queries

		dbsql := es.databaseCreateTableSql(event)

		if len(dbsql) > 0 {
			// Case matters for Sqlite3!
			// For example, code that works fine
			// when a table has lower-case field names
			// will fail if that same table
			// has camel case field names.
			// So, we use case-sensitive comparison for SQL.
			if q.Create != dbsql {
				msgfmt := "Table exists for %T, but SQL differs: '%s' != '%s'"
				panic(fmt.Sprintf(msgfmt, event, dbsql, q.Create))
			}
		} else {
			fmt.Printf("cqrs: creating schema in %s\n", es.datasource)
			es.db.MustExec(q.Create)
			es.db.MustExec(q.CreateIndexAggregateId)
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

// LoadEventsFor opens the gob file for the aggregator and returns any events found.
// If the file does not exist, an empty list is returned.
func (es *SqliteEventStore) LoadEventsFor(agg Aggregator) ([]Event, error) {
	return es.getEvents(agg.ID())
}

// BUG(mbucc) Overflows on a 32-bit system with 2,147,483,647 events.
func (es *SqliteEventStore) count(id AggregateID) int {
	var n, total int
	var err error
	var q string

	for _, info := range es.eventinfo {
		if isZero(id) {
			q = info.queries.CountAll
			err = es.db.Get(&n, q)
		} else {
			q = info.queries.Count
			err = es.db.Get(&n, q, id)
		}
		if err != nil {
			panic(fmt.Sprintf("cqrs: error running '%s', %v", q, err))
		}
		total += n
	}

	return total
}

func isZero(id AggregateID) bool {
	return reflect.ValueOf(id).Interface() == reflect.Zero(reflect.TypeOf(id)).Interface()
}

func (es *SqliteEventStore) GetAllEvents() ([]Event, error) {
	var id AggregateID
	return es.getEvents(id)
}

func (es *SqliteEventStore) getEvents(id AggregateID) ([]Event, error) {
	var err error

	n := es.count(id)

	events := make([]Event, n, n)

	i := 0
	for typ, info := range es.eventinfo {
		q := info.queries.Select
		if isZero(id) {
			q = info.queries.SelectAll
		}

		ptr := reflect.New(reflect.SliceOf(typ))
		iface := ptr.Interface()

		if isZero(id) {
			err = es.db.Select(iface, q)
		} else {
			err = es.db.Select(iface, q, id)
		}

		if err != nil {
			panic(fmt.Sprintf("cqrs: error running '%s', %v", q, err))
		}

		slce := ptr.Elem()
		for j := 0; j < slce.Len(); j++ {
			events[i] = slce.Index(j).Interface().(Event)
			i++
		}

	}

	return events, nil
}

// SaveEventsFor persists the events to disk for the given Aggregate.
func (es *SqliteEventStore) SaveEventsFor(agg Aggregator, loaded []Event, result []Event) error {
	for _, event := range result {
		info, ok := es.eventinfo[reflect.TypeOf(event)]
		if !ok {
			panic(fmt.Sprintf("cqrs: tried to save an event type (%T) that was not registered", event))
		}
		q := info.queries.Insert
		_, err := es.db.NamedExec(q, event)
		if err != nil {
			panic(fmt.Sprintf("cqrs: insert sql failed (%s) with event %+v: %v", q, event, err))
		}
	}
	return nil
}
