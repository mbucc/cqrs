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
	id        AggregateId
	Something string
}

func (e *HeardIt) Id() AggregateId {
	return e.id
}

type ShoutOutHandler struct{}

func (eh *ShoutOutHandler) handle(c Command) (a []Event, err error) {
	a = make([]Event, 1)
	c1 := c.(*ShoutOut)
	a[0] = &HeardIt{c1.Id(), c1.Comment}
	return a, nil
}

func TestHandledCommandReturnsEvents(t *testing.T) {

	Convey("Given a shout out and a shout out handler", t, func() {

		shout := ShoutOut{1, "ab"}
		h := ShoutOutHandler{}

		Convey("When the shout out is handled", func() {

			rval, _ := h.handle(&shout)

			Convey("It should return one event", func() {

				So(len(rval), ShouldEqual, 1)
			})
		})
	})
}

func TestAddHandlerFor(t *testing.T) {

	Convey("Given a message dispatcher and ShoutOut handler", t, func() {
		md := new(MessageDispatcher)

		Convey("When we register it as a handler for a ShoutOut command", func() {

			md.AddHandlerFor(new(ShoutOut), new(ShoutOutHandler))

			Convey("We can dispatch a ShoutOut event", func() {
				err := md.SendCommand(&ShoutOut{1, "Yo, Adrian!"})
				So(err, ShouldEqual, nil)
			})
		})
	})
}
