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
	"encoding/gob"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const aggFilenamePrefix string = "aggregate"

// Store events in file system in gob format.
// Each aggregate's event history
// is stored in a separate file.
//
// This event store has exponential behavior;
// at 50,000 events in history it can process
// 130 events/second.
// When history is only 25,000 events,
// it can process 500 events/second.
//
type GobEventStore struct {
	// BUG(mbucc) Modifying the RootDir of a GobEventStore will break things.
	RootDir       string
	EventsInStore uint64
}

// SetEventTypes registers event types
// so we can reconsitute into an interface.
// Will panic if the same eventype appears more than once.
func (es *GobEventStore) SetEventTypes(types []Event) error {
	for _, event := range types {
		gob.Register(event)
	}
	return nil
}

// Generate the file name used for the gob file for this aggregate.
func (es *GobEventStore) FileNameFor(agg Aggregator) string {
	t := fmt.Sprintf("%T", agg)
	if strings.HasPrefix(t, "*") {
		t = t[1:]
	}
	return fmt.Sprintf("%s/%s-%v_%v.gob", es.RootDir, aggFilenamePrefix, t, agg.ID())
}

// LoadEventsFor opens the gob file for the aggregator and returns any events found.
// If the file does not exist, an empty list is returned.
func (es *GobEventStore) LoadEventsFor(agg Aggregator) ([]Event, error) {
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

func (es *GobEventStore) GetAllEvents() ([]Event, error) {
	var events []Event = make([]Event, es.EventsInStore)
	gobfiles, err := filepath.Glob(fmt.Sprintf("%s/%s-*.gob", es.RootDir, aggFilenamePrefix))
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

func filenameToEvents(fn string) ([]Event, error) {
	var events []Event
	var event Event

	fp, err := os.Open(fn)
	if err != nil {
		return nil, fmt.Errorf("cqrs: can't read events from %s, %v", fn, err)
	}
	defer fp.Close()
	dec := gob.NewDecoder(fp)
	for {
		event = nil
		if err := dec.Decode(&event); err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, fmt.Errorf("cqrs: can't decode event in %s, %v", fn, err)
			}
		}
		events = append(events, event)
	}
	return events, nil
}

// SaveEventsFor persists the events to disk for the given Aggregate.
func (es *GobEventStore) SaveEventsFor(agg Aggregator, loaded []Event, result []Event) error {
	fn := es.FileNameFor(agg)
	tmpfn := fmt.Sprintf("%s.tmp", fn)
	fp, err := os.OpenFile(tmpfn, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("SaveEventsFor(%+v): can't open '%s', %v", agg, tmpfn, err)
	}
	enc := gob.NewEncoder(fp)
	for _, e := range append(loaded, result...) {
		if err := enc.Encode(&e); err != nil {
			fp.Close()
			return fmt.Errorf("SaveEventsFor(%+v): can't encode to '%s', %v", agg, fn, err)
		}
	}
	fp.Close()
	if err := os.Rename(tmpfn, fn); err != nil {
		return fmt.Errorf("SaveEventsFor(%+v): can't rename '%s' to %s, %v", tmpfn, fn, err)
	}
	return nil
}
