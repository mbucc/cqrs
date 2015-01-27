package cqrs

import (
	. "github.com/smartystreets/goconvey/convey"
//	"reflect"
	"testing"
)

var testChannel = make(chan string)

type ShoutCommand struct {
	id      AggregateID
	Comment string
}

func (c *ShoutCommand) ID() AggregateID {
	return c.id
}

type HeardEvent struct {
	id    AggregateID
	Heard string
}

func (e *HeardEvent) ID() AggregateID {
	return e.id
}

type EchoAggregate struct{
	id	AggregateID
}

func (eh *EchoAggregate) Handle(c Command) (a []Event, err error) {
	a = make([]Event, 1)
	c1 := c.(*ShoutCommand)
	a[0] = &HeardEvent{c1.ID(), c1.Comment}
	return a, nil
}

func (eh *EchoAggregate) ApplyEvents([]Event) {
}

func (eh *EchoAggregate) ID() AggregateID {
	return eh.id
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

		Convey("A ShoutCommand persist an event", func() {
			err := SendCommand(&ShoutCommand{aggid, "hello humanoid"})
			So(err, ShouldEqual, nil)
			events, err := store.LoadEventsFor(aggid)
			So(len(events), ShouldEqual, 1)
		})
	})
}

