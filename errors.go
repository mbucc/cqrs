/*
	Reasons why commands are rejected.

Since: Sun Nov 30 20:57:50 EST 2014
Author: Mark Bucciarelli <mkbucc@gmail.com>

*/
package main

type TabNotOpen int

func (i TabNotOpen) Error() string { return "TabNotOpen" }

type DrinksNotOutstanding int

func (i DrinksNotOutstanding) Error() string { return "DrinksNotOutstanding" }

type FoodNotOutstanding int

func (i FoodNotOutstanding) Error() string { return "FoodNotOutstanding" }

type FoodNotPrepared int

func (i FoodNotPrepared) Error() string { return "FoodNotPrepared" }

type MustPayEnough int

func (i MustPayEnough) Error() string { return "MustPayEnough" }

type TabHasUnservedItems int

func (i TabHasUnservedItems) Error() string { return "TabHasUnservedItems" }
