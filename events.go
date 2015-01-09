package main

type DrinksOrdered struct {
	id		TabId
	Items []OrderedItem
}

type DrinksServed struct {
	id		TabId
	MenuNumbers []int
}

type FoodOrdered struct {
	id		TabId
	Items []OrderedItem
}

type FoodPrepared struct {
	id		TabId
	MenuNumbers []int
}

type FoodServed struct {
	id		TabId
	MenuNumbers []int
}

type TabClosed struct {
	id		TabId
	AmountPaid float64
	OrderValue float64
	TipValue   float64
}

type TabOpened struct {
	id		TabId
	TableNumber int
	Waiter      string
}
