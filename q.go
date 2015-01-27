package cqrs

import (
	"encoding/gob"
	"fmt"
	"os"
)

// An event is something that happened
// as a result of a command;
// for example, FaceSlapped.
type Event interface {
	ID() AggregateID
}

// React to something that happened.
// Build a read model, send an email, whatever.
type EventListener interface {
	apply(e Event) error
}

// Persist and restore events.
type EventStorer interface {
	LoadEventsFor(AggregateID) ([]Event, error)
	SaveEventsFor(AggregateID, []Event, []Event) error
}

// An event storer that neither stores nor restores.
// It minimally satisfies the interface.
type NullEventStorer struct {}

func (es *NullEventStorer) LoadEventsFor(id AggregateID) ([]Event, error) {
	return []Event{}, nil
}

func (es *NullEventStorer) SaveEventsFor(id AggregateID, loaded []Event, result []Event) error {
	return nil
}

// Store events in file system.
// Events are stored in a file named
// <aggregate_type>_<aggregate_id>.gob
// and the events are stored as JSON.
type fileSystemEventStorer struct {
	rootdir	string
	eventTypes []Event
}

func NewFileSystemEventStorer(rootdir string, types []Event)  *fileSystemEventStorer {
	fes := new(fileSystemEventStorer)
	fes.rootdir = rootdir
	for _, event := range types {
		gob.Register(event)
	}
	return fes
}


func (es *fileSystemEventStorer) aggregateFileName(id AggregateID) string {
	return fmt.Sprintf("%s/aggregate%v.gob", es.rootdir, id)
}

func (es *fileSystemEventStorer) LoadEventsFor(id AggregateID) ([]Event, error) {
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

func (es *fileSystemEventStorer) SaveEventsFor(id AggregateID, loaded []Event, result []Event) error {
	fn := es.aggregateFileName(id)
	tmpfn := fn + ".tmp"

	// O_CREATE | O_EXCL is atomic (at least on POSIX systems)
	// so it's a way of ensuring only one goroutine
	// ever updates this aggregate.
	fp, err := os.OpenFile(tmpfn, os.O_CREATE | os.O_EXCL | os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("SaveEventsFor(%v): can't open '%s', %s", id, fn, err)
	}
	defer fp.Close()
	if err := gob.NewEncoder(fp).Encode(result); err != nil {
		return fmt.Errorf("SaveEventsFor(%v): can't encode to '%s', %s", id, tmpfn, err)
	}
	if err := os.Rename(tmpfn, fn); err != nil {
		return fmt.Errorf("SaveEventsFor(%v): rename failed '%s'-->'%s', %s", id, tmpfn, fn, err)
	}
	return nil
}
