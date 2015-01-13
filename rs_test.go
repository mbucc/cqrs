package main

import (
	. "github.com/smartystreets/goconvey/convey"
	"reflect"
	"testing"
	"log"
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

type SayItHysterically struct{}

func (h *SayItHysterically) apply(e Event) error {
	log.Print("SayItHysterically stub")
	return nil
}

type SayItBubbly struct{}

func (h *SayItBubbly) apply(e Event) error {
	log.Printf("SayItBubbly stub")
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
		registry := HandlerRegistry{
			reflect.TypeOf(new(ShoutOut)): new(EchoHandler)}
		md, err := NewMessageDispatcher(registry, nil)
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

	Convey("Given an echo handler and a couple SayIt listeners", t, func() {
		listeners := ListenerRegistry{
			reflect.TypeOf(new(HeardIt)):
				[]EventListener{new(SayItHysterically), new(SayItBubbly)}}
		registry := HandlerRegistry{
				reflect.TypeOf(new(ShoutOut)): new(EchoHandler)}
		md, err := NewMessageDispatcher(registry, listeners)
		So(err, ShouldEqual, nil)

		Convey("A ShoutOut should make noise", func() {
			events, err := md.SendCommand(&ShoutOut{1, "hello humanoid"})
			So(err, ShouldEqual, nil)
			So(len(events), ShouldEqual, 1)
			md.PublishEvent(events[0])
		})
	})
}
