/*
   Copyright (c) 2013, Edument AB
    Copyright (c) 2015, Mark Bucciarelli <mkbucc@gmail.com>

    Permission to use, copy, modify, and/or distribute this software
    for any purpose with or without fee is hereby granted, provided
    that the above copyright notice and this permission notice
    appear in all copies.

    THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL
    WARRANTIES WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED
    WARRANTIES OF MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL
    THE AUTHOR BE LIABLE FOR ANY SPECIAL, DIRECT, INDIRECT, OR
    CONSEQUENTIAL DAMAGES OR ANY DAMAGES WHATSOEVER RESULTING FROM
    LOSS OF USE, DATA OR PROFITS, WHETHER IN AN ACTION OF CONTRACT,
    NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF OR IN
    CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

package cqrs

// A NullEventStore is an event storer that neither stores nor restores.
// It minimally satisfies the interface.
type NullEventStore struct{}

// LoadEventsFor in the null EventStorer returns an empty array.
func (es *NullEventStore) LoadEventsFor(agg Aggregator) ([]Event, error) {
	return []Event{}, nil
}

// SaveEventsFor in the null EventStorer doesn't save anything.
func (es *NullEventStore) SaveEventsFor(agg Aggregator, loaded []Event, result []Event) error {
	return nil
}

// GetAllEvents in the null EventStorer doesn't load anything.
func (es *NullEventStore) GetAllEvents() ([]Event, error) {
	return []Event{}, nil
}

// SetEventTypes does nothing in the null event store.
func (es *NullEventStore) SetEventTypes(a []Event) {
}
