package main_test

import (
    "testing"
    . "github.com/mbucc/cqrs-sample-app"
    . "github.com/smartystreets/goconvey/convey"
)

type ShoutOut struct {
	Id		TabId
	Comment string
}

type HeardIt struct {
	Id		TabId
	Something	string
}

type ShoutHandler struct {}

func (eh *ShoutHandler) handle(c *ShoutOut) (e []Event, err error) {
	e = make([]Event, 1)
	e[0] = HeardIt{c.Id, c.Comment}
	return
}



func TestEchoCommand(t *testing.T) {
	Convey("Given a shout out and a shout out handler", t, func() {
		shout := ShoutOut{1, "ab"}
		h := ShoutHandler{};

		Convey("When the shout out is handled", func() {

			rval, _ := h.handle(&shout)

			Convey("It should return two events", func() {

				So(len(rval), ShouldEqual, 1)
			})
		})
	})
}
