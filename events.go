package cqrs

import (
	"encoding/gob"
	"fmt"
	"os"
)

type ErrConcurrency struct {
	eventCountNow    int
	eventCountStart int
	aggregateID      AggregateID
	newEvents        []Event
}

func (e *ErrConcurrency) Error() string {
	return fmt.Sprintf("cqrs: concurrency violation for aggregate id %v, %d (start) != %d (now)",
		e.aggregateID, e.eventCountStart, e.eventCountNow)
}

// An Event is something that happened
// as a result of a command;
// for example, FaceSlapped.
type Event interface {
	ID() AggregateID
}

// An EventListener is typically a read model,
// for example, a denormalized summary of your
// data that is very fast to query.
type EventListener interface {
	apply(e Event) error
}

// An EventStorer is an interface that defines the methods
// that persist events
type EventStorer interface {
	SetEventTypes([]Event)
	LoadEventsFor(AggregateID) ([]Event, error)
	SaveEventsFor(AggregateID, []Event, []Event) error
}

// A NullEventStore is an event storer that neither stores nor restores.
// It minimally satisfies the interface.
type NullEventStore struct{}

// LoadEventsFor in the null EventStorer returns an empty array.
func (es *NullEventStore) LoadEventsFor(id AggregateID) ([]Event, error) {
	return []Event{}, nil
}

// SaveEventsFor in the null EventStorer doesn't save anything.
func (es *NullEventStore) SaveEventsFor(id AggregateID, loaded []Event, result []Event) error {
	return nil
}

// SetEventTypes does nothing in the null event store.
func (es *NullEventStore) SetEventTypes(a []Event) {
}

// Store events in file system.
// Events are stored in a file named
// aggregate<aggregate_id>.gob
// and the events are stored as gob.
//
// BUG(mbucc) File names will collide if two different aggregate types have the same ID.
type FileSystemEventStore struct {
	rootdir    string
}

// Register event types so we can reconsitute into an interface.
// Will panic if the same eventype appears more than once.
func (fes *FileSystemEventStore) SetEventTypes(types []Event) {
	for _, event := range types {
		gob.Register(event)
	}
}

func (es *FileSystemEventStore) aggregateFileName(id AggregateID) string {
	return fmt.Sprintf("%s/aggregate%v.gob", es.rootdir, id)
}

// Load events from disk for the given aggregate.
func (es *FileSystemEventStore) LoadEventsFor(id AggregateID) ([]Event, error) {
	var events []Event
	fn := es.aggregateFileName(id)
	if _, err := os.Stat(fn); err != nil {
		if os.IsNotExist(err) {
			return events, nil
		}
		return nil, fmt.Errorf("LoadEventsFor(%v): can't stat '%s', %s", id, fn, err)
	}
	fp, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("LoadEventsFor(%v): can't open '%s', %s", id, fn, err)
	}
	defer fp.Close()
	decoder := gob.NewDecoder(fp)
	if err := decoder.Decode(&events); err != nil {
		return nil, fmt.Errorf("LoadEventsFor(%v): can't decode '%s', %s", id, fn, err)
	}
	return events, nil
}

// SaveEventsFor persists the events to disk for the given Aggregate.
func (es *FileSystemEventStore) SaveEventsFor(id AggregateID, loaded []Event, result []Event) error {
	fn := es.aggregateFileName(id)
	tmpfn := fn + ".tmp"

	if currentEvents, err := es.LoadEventsFor(id); err == nil {
		if len(currentEvents) != len(loaded) {
			return &ErrConcurrency{
				len(loaded), len(currentEvents), id, result}
		}
	} else {
		return fmt.Errorf("filesystem: can't get current contents of '%s', %s", fn, err)
	}

	// O_CREATE | O_EXCL is atomic (at least on POSIX systems)
	// so it ensures only one goroutine
	// ever updates this aggregate at a time.
	fp, err := os.OpenFile(tmpfn, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("SaveEventsFor(%v): can't open '%s', %s", id, fn, err)
	}
	defer fp.Close()
	if err := gob.NewEncoder(fp).Encode(append(loaded, result...)); err != nil {
		return fmt.Errorf("SaveEventsFor(%v): can't encode to '%s', %s", id, tmpfn, err)
	}
	if err := os.Rename(tmpfn, fn); err != nil {
		return fmt.Errorf("SaveEventsFor(%v): rename failed '%s'-->'%s', %s", id, tmpfn, fn, err)
	}
	return nil
}
