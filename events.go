package main

/*

	An event is something emitted by an accepted command.

	Queries, or ReadModels, may listen to events so they can
	update their own view-specific dataset.

Since : Sun Nov 30 21:52:26 EST 2014
Author: Mark Bucciarelli <mkbucc@gmail.com>

*/

import "github.com/twinj/uuid"

type Event interface {
	Id() string
}

type BaseEvent struct {
	id uuid.UUID
}

func (e *BaseEvent) Id() string {
	return e.id.String()
}

type DrinksOrdered struct {
	*BaseEvent
	Items []OrderedItem
}

type DrinksServed struct {
	*BaseEvent
	MenuNumbers []int
}

type FoodOrdered struct {
	*BaseEvent
	Items []OrderedItem
}

type FoodPrepared struct {
	*BaseEvent
	MenuNumbers []int
}

type FoodServed struct {
	*BaseEvent
	MenuNumbers []int
}

type TabClosed struct {
	*BaseEvent
	AmountPaid float64
	OrderValue float64
	TipValue   float64
}

type TabOpened struct {
	*BaseEvent
	TableNumber int
	Waiter      string
}
