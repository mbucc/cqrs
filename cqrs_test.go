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

package cqrs

import (
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

var testChannel = make(chan string)

type ShoutCommand struct {
	id               AggregateID
	Comment          string
	supportsRollback bool
}

func (c *ShoutCommand) ID() AggregateID {
	return c.id
}

func (c *ShoutCommand) BeginTransaction() error {
	return nil
}

func (c *ShoutCommand) Commit() error {
	return nil
}

func (c *ShoutCommand) Rollback() error {
	return nil
}
func (c *ShoutCommand) SupportsRollback() bool {
	return c.supportsRollback
}

type HeardEvent struct {
	BaseEvent
	id    AggregateID
	Heard string
}

func (e *HeardEvent) ID() AggregateID {
	return e.id
}

type EchoAggregate struct {
	id AggregateID
}

func (eh EchoAggregate) Handle(c Command) (a []Event, err error) {
	a = make([]Event, 1)
	c1 := c.(*ShoutCommand)
	a[0] = &HeardEvent{id: c1.ID(), Heard: c1.Comment}
	return a, nil
}

func (eh EchoAggregate) ID() AggregateID {
	return eh.id
}

func (eh EchoAggregate) New(id AggregateID) Aggregator {
	return &EchoAggregate{id}
}

func (eh EchoAggregate) ApplyEvents([]Event) {
}

type NullAggregate struct {
	id AggregateID
}

func (eh NullAggregate) Handle(c Command) (a []Event, err error) {
	a = make([]Event, 1)
	c1 := c.(*ShoutCommand)
	a[0] = &HeardEvent{id: c1.ID(), Heard: c1.Comment}
	return a, nil
}

func (eh NullAggregate) ID() AggregateID {
	return eh.id
}

func (eh NullAggregate) New(id AggregateID) Aggregator {
	return &NullAggregate{id}
}

func (eh NullAggregate) ApplyEvents([]Event) {
}

type SlowDownEchoAggregate struct {
	id AggregateID
}

func (h SlowDownEchoAggregate) ID() AggregateID {
	return h.id
}
func (eh SlowDownEchoAggregate) New(id AggregateID) Aggregator {
	return &SlowDownEchoAggregate{id}
}

func (h SlowDownEchoAggregate) Handle(c Command) (a []Event, err error) {
	a = make([]Event, 1)
	c1 := c.(*ShoutCommand)
	if strings.HasPrefix(c1.Comment, "slow") {
		time.Sleep(250 * time.Millisecond)
	}
	a[0] = &HeardEvent{id: c1.ID(), Heard: c1.Comment}
	return a, nil
}

func (h SlowDownEchoAggregate) ApplyEvents([]Event) {
}

type ChannelWriterEventListener struct{}

func (h *ChannelWriterEventListener) Apply(e Event) error {
	testChannel <- e.(*HeardEvent).Heard
	return nil
}

func (h *ChannelWriterEventListener) Reapply(e Event) error {
	return nil
}

type NullEventListener struct{}

func (h *NullEventListener) Apply(e Event) error {
	return nil
}

func (h *NullEventListener) Reapply(e Event) error {
	return nil
}

func TestHandledCommandReturnsEvents(t *testing.T) {

	Convey("Given a shout out and a shout out handler", t, func() {

		shout := ShoutCommand{1, "ab", false}
		h := EchoAggregate{}

		Convey("When the shout out is handled", func() {

			rval, _ := h.Handle(&shout)

			Convey("It should return one event", func() {

				So(len(rval), ShouldEqual, 1)
			})
		})
	})
}

func TestSendCommand(t *testing.T) {

	unregisterAll()

	Convey("Given an echo handler and two channel writerlisteners", t, func() {

		RegisterEventListeners(new(HeardEvent),
			new(ChannelWriterEventListener),
			new(ChannelWriterEventListener))
		RegisterCommandAggregator(new(ShoutCommand), EchoAggregate{})
		RegisterEventStore(new(NullEventStore))
		Convey("A ShoutCommand should be heard", func() {
			go func() {
				Convey("SendCommand should succeed", t, func() {
					err := SendCommand(&ShoutCommand{1, "hello humanoid", false})
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

func TestFileSystemEventStore(t *testing.T) {

	unregisterAll()

	aggid := AggregateID(1)
	agg := EchoAggregate{aggid}
	store := &FileSystemEventStore{RootDir: "/tmp"}
	RegisterEventListeners(new(HeardEvent), new(NullEventListener))
	RegisterEventStore(store)
	RegisterCommandAggregator(new(ShoutCommand), EchoAggregate{})

	Convey("Given an echo handler and two null listeners", t, func() {

		Convey("A ShoutCommand should persist an event", func() {
			err := SendCommand(&ShoutCommand{aggid, "hello humanoid", false})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(agg)
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 1)
		})
		Reset(func() {
			os.Remove(store.FileNameFor(agg))
		})
	})
}

func TestFileStorePersistsOldAndNewEvents(t *testing.T) {

	unregisterAll()

	Convey("Given an echo handler and two null listeners", t, func() {

		aggid := AggregateID(1)
		agg := EchoAggregate{aggid}
		store := &FileSystemEventStore{RootDir: "/tmp"}
		RegisterEventListeners(new(HeardEvent), new(NullEventListener))
		RegisterEventStore(store)
		RegisterCommandAggregator(new(ShoutCommand), EchoAggregate{})

		Convey("A ShoutCommand should persist old and new events", func() {
			err := SendCommand(&ShoutCommand{aggid, "hello humanoid", false})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(agg)
			So(len(events), ShouldEqual, 1)

			err = SendCommand(&ShoutCommand{aggid, "hello humanoid", false})
			So(err, ShouldEqual, nil)
			events, err = store.LoadEventsFor(agg)
			So(len(events), ShouldEqual, 2)
		})
		Reset(func() {
			os.Remove(store.FileNameFor(agg))
		})
	})
}

func TestConcurrencyError(t *testing.T) {

	unregisterAll()

	Convey("Given a fast/slow echo handler, a null listener, and a file system store", t, func() {

		store := &FileSystemEventStore{RootDir: "/tmp"}
		RegisterEventListeners(new(HeardEvent), new(NullEventListener))
		RegisterEventStore(store)
		RegisterCommandAggregator(new(ShoutCommand), SlowDownEchoAggregate{})
		agg := SlowDownEchoAggregate{1}

		Convey("Given one slow and then one fast echo", func() {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				Convey("The slow echo should succeed on retry", t, func() {
					err := SendCommand(&ShoutCommand{1, "slow hello humanoid", false})
					So(err, ShouldNotEqual, nil)
				})
				wg.Done()
			}()
			// Sleep a bit to make sure previous
			// handler kicks off first.
			time.Sleep(100 * time.Millisecond)
			err := SendCommand(&ShoutCommand{1, "hello humanoid", false})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(agg)
			So(len(events), ShouldEqual, 1)
			wg.Wait()
		})
		Reset(func() {
			os.Remove(store.FileNameFor(agg))
			os.Remove(store.FileNameFor(agg) + ".tmp")
		})
	})
}

func TestRetryOnConcurrencyError(t *testing.T) {

	unregisterAll()

	Convey("Given a fast/slow echo handler, a null listener, and a file system store", t, func() {

		store := &FileSystemEventStore{RootDir: "/tmp"}
		RegisterEventListeners(new(HeardEvent), new(NullEventListener))
		RegisterEventStore(store)
		RegisterCommandAggregator(new(ShoutCommand), SlowDownEchoAggregate{})

		agg := SlowDownEchoAggregate{1}

		Convey("Given one slow and then one fast echo", func() {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				Convey("The slow echo should succeed on retry", t, func() {
					err := SendCommand(&ShoutCommand{1, "slow hello humanoid", true})
					So(err, ShouldEqual, nil)
				})
				wg.Done()
			}()
			// Sleep a bit to make sure previous
			// handler kicks off first.
			time.Sleep(100 * time.Millisecond)
			err := SendCommand(&ShoutCommand{1, "hello humanoid", false})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(agg)
			So(len(events), ShouldEqual, 1)
			wg.Wait()
		})
		Reset(func() {
			os.Remove(store.FileNameFor(agg))
			os.Remove(store.FileNameFor(agg) + ".tmp")
		})
	})
}

func TestReloadHistory(t *testing.T) {

	unregisterAll()

	Convey("Given an event history", t, func() {

		store := &FileSystemEventStore{RootDir: "/tmp"}
		RegisterEventListeners(new(HeardEvent), new(NullEventListener))
		RegisterEventStore(store)
		RegisterCommandAggregator(new(ShoutCommand), NullAggregate{})

		So(SendCommand(&ShoutCommand{1, "hello1", true}), ShouldEqual, nil)

		events, err := store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 1)

		So(SendCommand(&ShoutCommand{2, "hello2", true}), ShouldEqual, nil)
		So(SendCommand(&ShoutCommand{2, "hello2a", true}), ShouldEqual, nil)
		So(SendCommand(&ShoutCommand{3, "hello3", true}), ShouldEqual, nil)

		events, err = store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 4)

		Convey("We should be able to reload history", func() {
			unregisterAll()
			RegisterEventListeners(new(HeardEvent), new(NullEventListener))
			RegisterEventStore(store)
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
			os.Remove(store.FileNameFor(NullAggregate{1}))
			os.Remove(store.FileNameFor(NullAggregate{1}) + ".tmp")
			os.Remove(store.FileNameFor(NullAggregate{2}))
			os.Remove(store.FileNameFor(NullAggregate{2}) + ".tmp")
			os.Remove(store.FileNameFor(NullAggregate{3}))
			os.Remove(store.FileNameFor(NullAggregate{3}) + ".tmp")
		})
	})
}
func TestSequenceNumberCorrectAfterReload(t *testing.T) {

	unregisterAll()

	Convey("Given an event history", t, func() {

		store := &FileSystemEventStore{RootDir: "/tmp"}
		RegisterEventListeners(new(HeardEvent), new(NullEventListener))
		RegisterEventStore(store)
		RegisterCommandAggregator(new(ShoutCommand), NullAggregate{})

		So(SendCommand(&ShoutCommand{1, "hello1", true}), ShouldEqual, nil)
		So(SendCommand(&ShoutCommand{2, "hello2", true}), ShouldEqual, nil)

		events, err := store.GetAllEvents()
		So(err, ShouldEqual, nil)
		So(len(events), ShouldEqual, 2)

		Convey("If we reload history, event sequence number should keep counting where it left off", func() {
			unregisterAll()
			RegisterEventListeners(new(HeardEvent), new(NullEventListener))
			RegisterEventStore(store)
			RegisterCommandAggregator(new(ShoutCommand), NullAggregate{})
			So(SendCommand(&ShoutCommand{1, "hello1", true}), ShouldEqual, nil)
			events, err := store.GetAllEvents()
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 3)
			var lastId uint64 = 0
			for _, e := range events {
				So(lastId, ShouldBeLessThan, e.GetSequenceNumber())
				lastId = e.GetSequenceNumber()
			}
		})
		Reset(func() {
			os.Remove(store.FileNameFor(NullAggregate{1}))
			os.Remove(store.FileNameFor(NullAggregate{1}) + ".tmp")
			os.Remove(store.FileNameFor(NullAggregate{2}))
			os.Remove(store.FileNameFor(NullAggregate{2}) + ".tmp")
		})
	})
}
