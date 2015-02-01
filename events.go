package cqrs

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const aggFilenamePrefix string = "aggregate"

// A concurrency error occurs if,
// after an Aggregator has loaded old events from the event store
// and before it has persisted new events resulting from the command processing,
// another command of the same type comes in
// and completes it's processing.
//
// The check for a consistency error is simple: when writing new events to the store,
// we check that the number of events on file
// are the same as the number of events loaded
// when the command processing began.
type ErrConcurrency struct {
	eventCountNow   int
	eventCountStart int
	aggregate       Aggregator
	newEvents       []Event
}

func (e *ErrConcurrency) Error() string {
	return fmt.Sprintf("cqrs: concurrency violation for aggregate %v, %d (start) != %d (now)",
		e.aggregate, e.eventCountStart, e.eventCountNow)
}

// An Event is something that happened
// as a result of a command;
// for example, FaceSlapped.
type Event interface {
	ID() AggregateID

	// We count atomically and give each event
	// a unique number.
	// Numbers for events within a command are not
	// guaranteed to be sequential; the only guarantee
	// is that a later event will have a higher number
	// than an earlier event.
	SetSequenceNumber(uint64)
	SequenceNumber() uint64
}

type BySequenceNumber []Event

func (s BySequenceNumber) Len() int           { return len(s) }
func (s BySequenceNumber) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s BySequenceNumber) Less(i, j int) bool { return s[i].SequenceNumber() < s[j].SequenceNumber() }

// An EventListener is typically a read model,
// for example, an in-memory denormalized summary of your
// data that is very fast to query.
//
type EventListener interface {
	apply(e Event) error
	reapply(e Event) error
}

// An EventStorer is an interface that defines the methods
// that persist events
type EventStorer interface {
	SetEventTypes([]Event)
	LoadEventsFor(Aggregator) ([]Event, error)
	SaveEventsFor(Aggregator, []Event, []Event) error
	GetAllEvents() ([]Event, error)
}

// A NullEventStore is an event storer that neither stores nor restores.
// It minimally satisfies the interface.
type NullEventStore struct{}

// LoadEventsFor in the null EventStorer returns an empty array.
func (es *NullEventStore) LoadEventsFor(agg Aggregator) ([]Event, error) {
	return []Event{}, nil
}

// SaveEventsFor in the null EventStorer doesn't save anything.
func (es *NullEventStore) SaveEventsFor(agg Aggregator, loaded []Event, result []Event) error {
	return nil
}

// GetAllEvents in the null EventStorer doesn't load anything.
func (es *NullEventStore) GetAllEvents() ([]Event, error) {
	return []Event{}, nil
}

// SetEventTypes does nothing in the null event store.
func (es *NullEventStore) SetEventTypes(a []Event) {
}

// Store events in file system.
// Events are stored in a file named
// aggregate<aggregate_id>.gob
// and the events are stored as gob.
//
type FileSystemEventStore struct {
	rootdir     string
	historySize uint64
}

// SetEventTypes registers event types
// so we can reconsitute into an interface.
// Will panic if the same eventype appears more than once.
func (fes *FileSystemEventStore) SetEventTypes(types []Event) {
	for _, event := range types {
		gob.Register(event)
	}
}

// Generate the file name used for the gob file for this aggregate.
func (es *FileSystemEventStore) FileNameFor(agg Aggregator) string {
	t := fmt.Sprintf("%T", agg)
	if strings.HasPrefix(t, "*") {
		t = t[1:]
	}
	return fmt.Sprintf("%s/%s-%v_%v.gob", es.rootdir, aggFilenamePrefix, t, agg.ID())
}

func filenameToEvents(fn string) ([]Event, error) {
	var events []Event
	fp, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("cqrs: can't read events from %s, %v", fn, err)
	}
	defer fp.Close()
	decoder := gob.NewDecoder(fp)
	if err := decoder.Decode(&events); err != nil {
		return nil, fmt.Errorf("cqrs: can't decode events in %s, %v", fn, err)
	}
	return events, nil
}

// LoadEventsFor opens the gob file for the aggregator and returns any events found.
// If the file does not exist, an empty list is returned.
func (es *FileSystemEventStore) LoadEventsFor(agg Aggregator) ([]Event, error) {
	var events []Event
	fn := es.FileNameFor(agg)
	if _, err := os.Stat(fn); err != nil {
		if os.IsNotExist(err) {
			return events, nil
		}
		return nil, fmt.Errorf("LoadEventsFor(%v): can't stat '%s', %s", agg, fn, err)
	}
	return filenameToEvents(fn)
}

func (es *FileSystemEventStore) GetAllEvents() ([]Event, error) {
	var events []Event = make([]Event, es.historySize)
	gobfiles, err := filepath.Glob(fmt.Sprintf("%s/%s-*.gob", es.rootdir, aggFilenamePrefix))
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
func (es *FileSystemEventStore) SaveEventsFor(agg Aggregator, loaded []Event, result []Event) error {
	fn := es.FileNameFor(agg)
	tmpfn := fn + ".tmp"

	if currentEvents, err := es.LoadEventsFor(agg); err == nil {
		if len(currentEvents) != len(loaded) {
			return &ErrConcurrency{
				len(loaded), len(currentEvents), agg, result}
		}
	} else {
		return fmt.Errorf("filesystem: can't get current contents of '%s', %s", fn, err)
	}

	// O_CREATE | O_EXCL is atomic (at least on POSIX systems)
	// so it ensures only one goroutine
	// ever updates this aggregate at a time.
	fp, err := os.OpenFile(tmpfn, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("SaveEventsFor(%v): can't open '%s', %s", agg, fn, err)
	}
	defer fp.Close()
	if err := gob.NewEncoder(fp).Encode(append(loaded, result...)); err != nil {
		return fmt.Errorf("SaveEventsFor(%v): can't encode to '%s', %s", agg, tmpfn, err)
	}
	if err := os.Rename(tmpfn, fn); err != nil {
		return fmt.Errorf("SaveEventsFor(%v): rename failed '%s'-->'%s', %s", agg, tmpfn, fn, err)
	}
	return nil
}
