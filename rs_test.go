package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"log"
	"reflect"
	"testing"
)

var testChannel = make(chan string)

type ShoutOut struct {
	id      AggregateId
	Comment string
}

func (c *ShoutOut) Id() AggregateId {
	return c.id
}

type HeardIt struct {
	id    AggregateId
	Heard string
}

func (e *HeardIt) Id() AggregateId {
	return e.id
}

type EchoHandler struct{}

func (eh *EchoHandler) handle(c Command) (a []Event, err error) {
	a = make([]Event, 1)
	c1 := c.(*ShoutOut)
	a[0] = &HeardIt{c1.Id(), c1.Comment}
	return a, nil
}

func (eh *EchoHandler) ApplyEvents([]Event) {
}

type WriteToChannel struct{}

func (h *WriteToChannel) apply(e Event) error {
	testChannel <- e.(*HeardIt).Heard
	return nil
}

func TestHandledCommandReturnsEvents(t *testing.T) {

	Convey("Given a shout out and a shout out handler", t, func() {

		shout := ShoutOut{1, "ab"}
		h := EchoHandler{}

		Convey("When the shout out is handled", func() {

			rval, _ := h.handle(&shout)

			Convey("It should return one event", func() {

				So(len(rval), ShouldEqual, 1)
			})
		})
	})
}

func TestSendCommand(t *testing.T) {

	Convey("Given a dispatcher that sends ShoutOuts to an EchoHandler", t, func() {
		registry := Aggregators{
			reflect.TypeOf(new(ShoutOut)): new(EchoHandler)}
		md, err := NewMessageDispatcher(registry, nil, nil)
		So(err, ShouldEqual, nil)

		Convey("Sending a ShoutOut should return a single HeardIt event", func() {
			events, err := md.SendCommand(&ShoutOut{1, "hello world"})
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 1)
			So(events[0].(*HeardIt).Heard, ShouldEqual, "hello world")
		})
	})
}

func TestPublishEvent(t *testing.T) {
	log.Println("starting TestPublishEvent")

	var err0 error
	if err0 != nil {
		log.Panicf("can't open temp file")
	}

	Convey("Given an echo handler and a couple SayIt listeners", t, func() {
		listeners := EventListeners{
			reflect.TypeOf(new(HeardIt)): []EventListener{new(WriteToChannel), new(WriteToChannel)}}
		registry := Aggregators{
			reflect.TypeOf(new(ShoutOut)): new(EchoHandler)}
		md, err := NewMessageDispatcher(registry, listeners, nil)
		So(err, ShouldEqual, nil)

		Convey("A ShoutOut should make noise", func() {
			events, err := md.SendCommand(&ShoutOut{1, "hello humanoid"})
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 1)
			go func() {
				Convey("Should be able to publish events", t, func() {
					err := md.PublishEvent(events[0])
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
