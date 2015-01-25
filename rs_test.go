package main

import (
	. "github.com/smartystreets/goconvey/convey"

	"reflect"
	"testing"
)

var testChannel = make(chan string)

type ShoutCommand struct {
	id      AggregateId
	Comment string
}

func (c *ShoutCommand) Id() AggregateId {
	return c.id
}

type HeardEvent struct {
	id    AggregateId
	Heard string
}

func (e *HeardEvent) Id() AggregateId {
	return e.id
}

type EchoAggregate struct{}

func (eh *EchoAggregate) handle(c Command) (a []Event, err error) {
	a = make([]Event, 1)
	c1 := c.(*ShoutCommand)
	a[0] = &HeardEvent{c1.Id(), c1.Comment}
	return a, nil
}

func (eh *EchoAggregate) ApplyEvents([]Event) {
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

			rval, _ := h.handle(&shout)

			Convey("It should return one event", func() {

				So(len(rval), ShouldEqual, 1)
			})
		})
	})
}

func TestSendCommand(t *testing.T) {

	Convey("Given an echo handler and a couple SayIt listeners", t, func() {
		listeners := EventListeners{
			reflect.TypeOf(new(HeardEvent)): []EventListener{
				new(ChannelWriterEventListener), new(ChannelWriterEventListener)}}
		registry := Aggregators{
			reflect.TypeOf(new(ShoutCommand)): new(EchoAggregate)}
		md, err := NewMessageDispatcher(registry, listeners, new(NullEventStore))
		So(err, ShouldEqual, nil)

		Convey("A ShoutCommand should make noise", func() {
			go func() {
				Convey("SendCommand should succeed", t, func() {
					err := md.SendCommand(&ShoutCommand{1, "hello humanoid"})
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
	aggid := AggregateId(1)
	es := &FileSystemEventStore{"/tmp"}

	Convey("Given an echo handler and a couple SayIt listeners", t, func() {
		listeners := EventListeners{
			reflect.TypeOf(new(HeardEvent)): []EventListener{
				new(NullEventListener), new(NullEventListener)}}
		registry := Aggregators{
			reflect.TypeOf(new(ShoutCommand)): new(EchoAggregate)}
		md, err := NewMessageDispatcher(registry, listeners, es)
		So(err, ShouldEqual, nil)

		Convey("A ShoutCommand persist two events", func() {
			err := md.SendCommand(&ShoutCommand{aggid, "hello humanoid"})
			So(err, ShouldEqual, nil)
			So(len(es.LoadEventsFor(aggid)), ShouldEqual, 2)
		})
	})
}

