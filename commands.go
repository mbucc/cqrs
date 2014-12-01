/*

	A command is a request is either accepted or rejected.

	All validations and data updates occur as a result of a
	command that is accepted.  A command never returns data.
	Data is always returned by a Query.  Queries either can hit
	the database directly or listen to events emitted by a
	successful Command.

Since : Sun Nov 30 20:42:18 EST 2014
Author: Mark Bucciarelli <mkbucc@gmail.com>

*/

package main

// XXX: The OrderedItem type doesn't belongs here.
type OrderedItem struct {
	MenuNumber  int
	Description string
	IsDrink     bool
	Price       decimal
}

type Command interface {
	Id() Guid
}

type BaseCommand struct {
	id Guid
}

func (c *BaseCommand) Id() Guid {
	return c.id
}

type CloseTab struct {
	*BaseCommand
	AmountPaid float64
}

type MarkDrinksServed struct {
	*BaseCommand
	MenuNumbers []int
}

type MarkFoodPrepared struct {
	*BaseCommand
	MenuNumbers []int
}

type MarkFoodServed struct {
	*BaseCommand
	MenuNumbers []int
}

type OpenTab struct {
	*BaseCommand
	TableNumber int
	Waiter      string
}

type PlaceOrder struct {
	*BaseCommand
	Items []OrderedItem
}
