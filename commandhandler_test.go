package main_test

import (
	. "github.com/mbucc/cqrs-sample-app"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

type ShoutOut struct {
	id      AggregateId
	Comment string
}

type HeardIt struct {
	id        AggregateId
	Something string
}

func (e *HeardIt) Id() AggregateId {
	return e.id
}

type ShoutOutHandler struct{}

func (eh *ShoutOutHandler) handle(c *ShoutOut) (a []Event, err error) {
	a = make([]Event, 1)
	a[0] = &HeardIt{c.id, c.Comment}
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
