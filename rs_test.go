package cqrs

import (
	. "github.com/smartystreets/goconvey/convey"
	"os"
	"testing"
	"time"
)

var testChannel = make(chan string)

type ShoutCommand struct {
	id      AggregateID
	Comment string
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


type HeardEvent struct {
	id    AggregateID
	Heard string
}

func (e *HeardEvent) ID() AggregateID {
	return e.id
}

type EchoAggregate struct{
}

func (eh *EchoAggregate) Handle(c Command) (a []Event, err error) {
	a = make([]Event, 1)
	c1 := c.(*ShoutCommand)
	a[0] = &HeardEvent{c1.ID(), c1.Comment}
	return a, nil
}

func (eh *EchoAggregate) ApplyEvents([]Event) {
}


type SlowThenFastEchoAggregate struct{
	toggle  int
}

func (h *SlowThenFastEchoAggregate) Handle(c Command) (a []Event, err error) {
	h.toggle += 1
	if h.toggle % 2 != 0 {
		time.Sleep(250 * time.Millisecond)
	}
	a = make([]Event, 1)
	c1 := c.(*ShoutCommand)
	a[0] = &HeardEvent{c1.ID(), c1.Comment}
	return a, nil
}

func (eh *SlowThenFastEchoAggregate) ApplyEvents([]Event) {
}

type ChannelWriterEventListener struct{}

func (h *ChannelWriterEventListener) apply(e Event) error {
	testChannel <- e.(*HeardEvent).Heard
	return nil
}

type NullEventListener struct{}

func (h *NullEventListener) apply(e Event) error {
	return nil
}

func TestHandledCommandReturnsEvents(t *testing.T) {

	Convey("Given a shout out and a shout out handler", t, func() {

		shout := ShoutCommand{1, "ab"}
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
		RegisterCommand(new(ShoutCommand), new(EchoAggregate))
		RegisterEventStore(new(NullEventStore))
		Convey("A ShoutCommand should be heard", func() {
			go func() {
				Convey("SendCommand should succeed", t, func() {
					err := SendCommand(&ShoutCommand{1, "hello humanoid"})
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

func TestFileSystemEventStorer(t *testing.T) {

	unregisterAll()

	aggid := AggregateID(1)
	store := NewFileSystemEventStorer("/tmp", []Event{&HeardEvent{}})
	RegisterEventStore(store)
	RegisterEventListeners(new(HeardEvent), new(NullEventListener))
	RegisterCommand(new(ShoutCommand), new(EchoAggregate))

	Convey("Given an echo handler and two null listeners", t, func() {

		Convey("A ShoutCommand should persist an event", func() {
			err := SendCommand(&ShoutCommand{aggid, "hello humanoid"})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(aggid)
			So(len(events), ShouldEqual, 1)
		})
		Reset(func() {
			os.Remove("/tmp/aggregate1.gob")
		})
	})

}

func TestFileStorePersistsOldAndNewEvents(t *testing.T) {

	unregisterAll()

	Convey("Given an echo handler and two null listeners", t, func() {

		aggid := AggregateID(1)
		store := NewFileSystemEventStorer("/tmp", []Event{&HeardEvent{}})
		RegisterEventStore(store)
		RegisterEventListeners(new(HeardEvent), new(NullEventListener))
		RegisterCommand(new(ShoutCommand), new(EchoAggregate))

		Convey("A ShoutCommand should persist old and new events", func() {
			err := SendCommand(&ShoutCommand{aggid, "hello humanoid"})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(aggid)
			So(len(events), ShouldEqual, 1)

			err = SendCommand(&ShoutCommand{aggid, "hello humanoid"})
			So(err, ShouldEqual, nil)
			events, err = store.LoadEventsFor(aggid)
			So(len(events), ShouldEqual, 2)
			})
		Reset(func() {
			os.Remove("/tmp/aggregate1.gob")
		})
	})
}

func TestConcurrencyError(t *testing.T) {

	unregisterAll()

	Convey("Given a fast/slow echo handler, a null listener, and a file system store", t, func() {

		store := NewFileSystemEventStorer("/tmp", []Event{&HeardEvent{}})
		RegisterEventStore(store)
		RegisterEventListeners(new(HeardEvent), new(NullEventListener))
		RegisterCommand(new(ShoutCommand), new(EchoAggregate))

		Convey("Given one slow and then one fast echo", func() {
			go func() {
				Convey("The slow echo should fail", t, func() {
					err := SendCommand(&ShoutCommand{1, "hello humanoid"})
					So(err, ShouldNotEqual, nil)
				})
			}()
			err := SendCommand(&ShoutCommand{1, "hello humanoid"})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(1)
			So(len(events), ShouldEqual, 1)
			})
		Reset(func() {
			os.Remove("/tmp/aggregate1.gob")
			os.Remove("/tmp/aggregate1.gob.tmp")
		})
	})
}

