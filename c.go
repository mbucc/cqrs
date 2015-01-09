// A Go version of Edument's (http://cqrs.nu/) CQRS sample application.
// CQRS stands for command/query responsibility separation.
package main


// A TabId represents a tab, or bill, at a cafe table.
type TabId int

// A command is an action
// that can either be accepted or rejected
// A command is always associated with a tab;
// for example, MoreDrinksWench!
// might be a command in a medieval
// misogynistic kind of cafe.
type Command interface {}

// An event is something that actually happens.
// and are triggered by commands.
// For example, ASlapInTheFace is a possible
// event that might be triggered
// by the MoreDrinksWench! command.
type Event interface {}

// Command handlers are responsible
// for validating commands,
// both in as a stand-alone type
// as well as in the context
// of the Command's aggregate (I know,
// lots of undefined terms here ... 
// see the github wiki).
type CommandHandler interface {
        handle(c *Command) (e []Event)
}
