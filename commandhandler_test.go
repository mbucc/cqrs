package main_test

import (
	. "github.com/mbucc/cqrs-sample-app"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

type ShoutOut struct {
	Id      TabId
	Comment string
}

type HeardIt struct {
	Id        TabId
	Something string
}

type ShoutOutHandler struct{}

func (eh *ShoutOutHandler) handle(c *ShoutOut) (e []Event, err error) {
	e = make([]Event, 1)
	e[0] = HeardIt{c.Id, c.Comment}
	return
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
