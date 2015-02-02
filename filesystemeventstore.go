package cqrs

import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
