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
