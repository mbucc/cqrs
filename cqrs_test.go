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
	id             AggregateID
	Heard          string
	SequenceNumber uint64
}

func (e *HeardEvent) ID() AggregateID {
	return e.id
}

func (e *HeardEvent) SetSequenceNumber(n uint64) {
	e.SequenceNumber = n
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

func (h *ChannelWriterEventListener) apply(e Event) error {
	testChannel <- e.(*HeardEvent).Heard
	return nil
}

func (h *ChannelWriterEventListener) reapply(e Event) error {
	return nil
}

type NullEventListener struct{}

func (h *NullEventListener) apply(e Event) error {
	return nil
}

func (h *NullEventListener) reapply(e Event) error {
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
	store := &FileSystemEventStore{"/tmp"}
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
		store := &FileSystemEventStore{"/tmp"}
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

		store := &FileSystemEventStore{"/tmp"}
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

		store := &FileSystemEventStore{"/tmp"}
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
