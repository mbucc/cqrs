package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

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

func TestOnlyOneHandlerPerCommand(t *testing.T) {

	Convey("You can't register two handlers for the same command", t, func() {
		handlers := []HandlerPair{
			HandlerPair{new(ShoutOut), nil},
			HandlerPair{new(ShoutOut), nil}}
		_, err := NewMessageDispatcher(handlers)
		So(err, ShouldEqual, nil)
	})
}

func TestSendCommand(t *testing.T) {

	Convey("Given a dispatcher that sends ShoutOut's to a EchoHandler", t, func() {
		handlers := []HandlerPair{
			HandlerPair{new(ShoutOut), new(EchoHandler)}}
		md, err := NewMessageDispatcher(handlers)
		So(err, ShouldEqual, nil)

		Convey("Sending a ShoutOut should return a single HeardIt event", func() {
			events, err := md.SendCommand(&ShoutOut{1, "hello world"})
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 1)
			So(events[0].(*HeardIt).Heard, ShouldEqual, "hello world")
		})
	})
}
