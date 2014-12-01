/*

	A command is a request is either accepted or rejected.

	All validations and data updates occur as a result of a
	command that is accepted.  A command never returns data.

	Returning data is the responsibility of a query.  A query
	can either hit the database directly or listen to events
	emitted by a successful command and maintain their own
	view-specific data set to query.

Since : Sun Nov 30 20:42:18 EST 2014
Author: Mark Bucciarelli <mkbucc@gmail.com>

*/

package main

import "github.com/twinj/uuid"

type Command interface {
	Id() string
}

type BaseCommand struct {
	id uuid.UUID
}

func (c *BaseCommand) Id() string {
	return c.id.String()
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
