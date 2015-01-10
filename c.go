// A Go version of Edument's (http://cqrs.nu/) CQRS sample application.
// CQRS stands for command/query responsibility separation.
// I wanted to learn about command/query separation.
package main

// In the Edument sample app,
// each tab defines a distinct aggregate instance
// and all commands and events associated with that tab
// use the same id.
type AggregateId int

// A command is an action
// that can be accepted or rejected
// for example, MoreDrinksWench!
// might be a command in a medieval
// misogynistic kind of cafe.
type Command interface{
	Id()  AggregateId
}

// An event is something that happened
// as a result of a command;
// for example, FaceSlapped.
type Event interface{
	Id()	AggregateId
}

// Command handlers are responsible
// for validating commands,
// both as a stand-alone set of data
// as well as in the context
// of the Command's aggregate (I know,
// lots of undefined terms here ...
// see the github wiki).
type CommandHandler interface {
	handle(c Command) (e []Event, err error)
}
