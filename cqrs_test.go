/*
   Copyright (c) 2013, Edument AB
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

package cqrs_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mbucc/cqrs"
	. "github.com/smartystreets/goconvey/convey"
)

const (
	testgobdir = "/tmp"
	testdb     = "/tmp/testcqrs.db"
)

var testChannel = make(chan string)

type NullAggregate struct{ id cqrs.AggregateID }

func (h *NullAggregate) Handle(c cqrs.Command) (a []cqrs.Event, err error) {
	a = make([]cqrs.Event, 1)
	c1 := c.(*ShoutSomething)
	a[0] = &HeardSomething{BaseEvent: cqrs.BaseEvent{Id: c1.ID()}, Heard: c1.Comment}
	return a, nil
}

func (h *NullAggregate) ID() cqrs.AggregateID                    { return h.id }
func (h *NullAggregate) New(id cqrs.AggregateID) cqrs.Aggregator { return &NullAggregate{id} }
func (h *NullAggregate) ApplyEvents([]cqrs.Event)                {}

type SlowDownEchoAggregate struct{ id cqrs.AggregateID }

func (h *SlowDownEchoAggregate) ID() cqrs.AggregateID { return h.id }
func (h *SlowDownEchoAggregate) New(id cqrs.AggregateID) cqrs.Aggregator {
	return &SlowDownEchoAggregate{id}
}

func (h *SlowDownEchoAggregate) Handle(c cqrs.Command) (a []cqrs.Event, err error) {
	a = make([]cqrs.Event, 1)
	c1 := c.(*ShoutSomething)
	if strings.HasPrefix(c1.Comment, "slow") {
		time.Sleep(250 * time.Millisecond)
	}
	a[0] = &HeardSomething{BaseEvent: cqrs.BaseEvent{Id: c1.ID()}, Heard: c1.Comment}
	return a, nil
}

func (h *SlowDownEchoAggregate) ApplyEvents([]cqrs.Event) {}

type ChannelWriterEventListener struct{}

func (h *ChannelWriterEventListener) Apply(e cqrs.Event) error {
	testChannel <- e.(*HeardSomething).Heard
	return nil
}

func (h *ChannelWriterEventListener) Reapply(e cqrs.Event) error { return nil }

type NullEventListener struct{}

func (h *NullEventListener) Apply(e cqrs.Event) error   { return nil }
func (h *NullEventListener) Reapply(e cqrs.Event) error { return nil }

func ClearTestData() {
	if _, err := os.Stat(testdb); err == nil {
		if err := os.Remove(testdb); err != nil {
			panic(fmt.Sprintf("error removing test database: %v", err))
		}
	}
	files, err := filepath.Glob(testgobdir + "/*.gob")
	if err != nil {
		panic(fmt.Sprintf("error globbing gob filenames: %v", err))
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			panic(fmt.Sprintf("error removing gob file '%s': %v", f, err))
		}
	}
}

func TestHandledCommandReturnsEvents(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	Convey("Given a shout out and a shout out handler", t, func() {

		shout := ShoutSomething{1, "ab"}
		h := &EchoAggregate{}

		Convey("When the shout out is handled", func() {

			rval, _ := h.Handle(&shout)

			Convey("It should return one event", func() {

				So(len(rval), ShouldEqual, 1)
			})
		})
	})
}

func TestSendCommand(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	Convey("Given an echo handler and two channel writerlisteners", t, func() {

		cqrs.RegisterEventListeners(new(HeardSomething),
			new(ChannelWriterEventListener),
			new(ChannelWriterEventListener))
		cqrs.RegisterCommandAggregator(new(ShoutSomething), &EchoAggregate{})
		cqrs.RegisterEventStore(new(cqrs.NullEventStore))
		Convey("A ShoutSomething should be heard", func() {
			go func() {
				Convey("cqrs.SendCommand should succeed", t, func() {
					err := cqrs.SendCommand(&ShoutSomething{1, "hello humanoid"})
					So(err, ShouldEqual, nil)
				})
				close(testChannel)
			}()
			n := 0
			for {
				msg, channelOpen := <-testChannel
				if !channelOpen {
					break
				}
				n = n + 1
				So(msg, ShouldEqual, "hello humanoid")
			}
			So(n, ShouldEqual, 2)
		})
	})
}

func TestGobEventStore(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	aggid := cqrs.AggregateID(1)
	agg := &EchoAggregate{aggid}
	store := &cqrs.GobEventStore{RootDir: "/tmp"}
	cqrs.RegisterEventListeners(new(HeardSomething), new(NullEventListener))
	cqrs.RegisterEventStore(store)
	cqrs.RegisterCommandAggregator(new(ShoutSomething), &EchoAggregate{})

	Convey("Given an echo handler and two null listeners", t, func() {

		Convey("A ShoutSomething should persist an event", func() {
			err := cqrs.SendCommand(&ShoutSomething{aggid, "hello humanoid"})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(agg)
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 1)
		})
	})
}

func TestFileStorePersistsOldAndNewEvents(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	Convey("Given an echo handler and two null listeners", t, func() {

		aggid := cqrs.AggregateID(1)
		agg := &EchoAggregate{aggid}
		store := &cqrs.GobEventStore{RootDir: "/tmp"}
		cqrs.RegisterEventListeners(new(HeardSomething), new(NullEventListener))
		cqrs.RegisterEventStore(store)
		cqrs.RegisterCommandAggregator(new(ShoutSomething), &EchoAggregate{})

		Convey("A ShoutSomething should persist old and new events", func() {
			err := cqrs.SendCommand(&ShoutSomething{aggid, "hello humanoid1"})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(agg)
			So(len(events), ShouldEqual, 1)

			err = cqrs.SendCommand(&ShoutSomething{aggid, "hello humanoid2"})
			So(err, ShouldEqual, nil)
			events, err = store.LoadEventsFor(agg)
			So(len(events), ShouldEqual, 2)
		})
	})
}

func TestReloadHistory(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	Convey("Given an event history", t, func() {

		store := &cqrs.GobEventStore{RootDir: "/tmp"}
		cqrs.RegisterEventListeners(new(HeardSomething), new(NullEventListener))
		cqrs.RegisterEventStore(store)
		cqrs.RegisterCommandAggregator(new(ShoutSomething), &NullAggregate{})

		So(cqrs.SendCommand(&ShoutSomething{1, "hello1"}), ShouldEqual, nil)

		events, err := store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 1)

		So(cqrs.SendCommand(&ShoutSomething{2, "hello2"}), ShouldEqual, nil)
		So(cqrs.SendCommand(&ShoutSomething{2, "hello2a"}), ShouldEqual, nil)
		So(cqrs.SendCommand(&ShoutSomething{3, "hello3"}), ShouldEqual, nil)

		events, err = store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 4)

		Convey("We should be able to reload history", func() {
			cqrs.UnregisterAll()
			cqrs.RegisterEventListeners(new(HeardSomething), new(NullEventListener))
			cqrs.RegisterEventStore(store)
			events, err := store.GetAllEvents()
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 4)
			Convey("We should be able to reload history", func() {
				var lastId uint64 = 0
				for _, e := range events {
					So(lastId, ShouldBeLessThan, e.GetSequenceNumber())
					lastId = e.GetSequenceNumber()
				}
			})
		})
		Reset(func() {
			ClearTestData()
		})
	})
}

func TestSequenceNumberCorrectAfterReload(t *testing.T) {

	cqrs.UnregisterAll()
	ClearTestData()

	Convey("Given an event history", t, func() {

		store := &cqrs.GobEventStore{RootDir: "/tmp"}
		cqrs.RegisterEventListeners(new(HeardSomething), new(NullEventListener))
		cqrs.RegisterEventStore(store)
		cqrs.RegisterCommandAggregator(new(ShoutSomething), &NullAggregate{})

		So(cqrs.SendCommand(&ShoutSomething{1, "hello1"}), ShouldEqual, nil)
		So(cqrs.SendCommand(&ShoutSomething{2, "hello2"}), ShouldEqual, nil)

		events, err := store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 2)

		Convey("If we reload history, event sequence number should keep counting where it left off", func() {
			cqrs.UnregisterAll()
			cqrs.RegisterEventListeners(new(HeardSomething), new(NullEventListener))
			cqrs.RegisterEventStore(store)
			cqrs.RegisterCommandAggregator(new(ShoutSomething), &NullAggregate{})
			So(cqrs.SendCommand(&ShoutSomething{1, "hello1"}), ShouldEqual, nil)
			events, err := store.GetAllEvents()
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 3)
			var lastId uint64 = 0
			for _, e := range events {
				So(lastId, ShouldBeLessThan, e.GetSequenceNumber())
				lastId = e.GetSequenceNumber()
			}
		})
	})
}
