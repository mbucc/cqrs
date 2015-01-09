package main

type CloseTab struct {
	Id         TabId
	AmountPaid float64
}

type MarkDrinksServed struct {
	Id          TabId
	MenuNumbers []int
}

type MarkFoodPrepared struct {
	Id          TabId
	MenuNumbers []int
}

type MarkFoodServed struct {
	Id          TabId
	MenuNumbers []int
}

type OpenTab struct {
	Id          TabId
	TableNumber int
	Waiter      string
}

type PlaceOrder struct {
	Id    TabId
	Items []OrderedItem
}
