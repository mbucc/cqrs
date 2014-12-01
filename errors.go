/*
	Reasons why commands are rejected.

Since: Sun Nov 30 20:57:50 EST 2014
Author: Mark Bucciarelli <mkbucc@gmail.com>

*/
package main

type  TabNotOpen            int
type  DrinksNotOutstanding  int
type  FoodNotOutstanding    int
type  FoodNotPrepared       int
type  MustPayEnough         int
type  TabHasUnservedItems   int

func  (i TabNotOpen)            Error()  string  { return  "TabNotOpen"           }
func  (i DrinksNotOutstanding)  Error()  string  { return  "DrinksNotOutstanding" }
func  (i FoodNotOutstanding)    Error()  string  { return  "FoodNotOutstanding"   }
func  (i FoodNotPrepared)       Error()  string  { return  "FoodNotPrepared"      }
func  (i MustPayEnough)         Error()  string  { return  "MustPayEnough"        }
func  (i TabHasUnservedItems)   Error()  string  { return  "TabHasUnservedItems"  }
