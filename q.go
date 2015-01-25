package cqrs

// An event is something that happened
// as a result of a command;
// for example, FaceSlapped.
type Event interface {
	Id() AggregateId
}

// React to something that happened.
// Build a read model, send an email, whatever.
type EventListener interface {
	apply(e Event) error
}

// Persist and restore events.
type EventStorer interface {
	LoadEventsFor(AggregateId) ([]Event, error)
	SaveEventsFor(AggregateId, []Event, []Event)
}

// An event storer that neither stores nor restores.
// It minimally satisfies the interface.
type NullEventStorer struct {}

func (es *NullEventStorer) LoadEventsFor(id AggregateId) ([]Event, error) {
	return []Event{}, nil
}

func (es *NullEventStorer) SaveEventsFor(id AggregateId, loaded []Event, result []Event) {
}

// Store events in file system.
// Events are stored in a file named
// <aggregate_type>_<aggregate_id>.json
// and the events are stored as JSON.
type FileSystemEventStorer struct {
	rootdir	string
}

func (es *FileSystemEventStorer) LoadEventsFor(id AggregateId) ([]Event, error) {
	return []Event{}, nil
}

func (es *FileSystemEventStorer) SaveEventsFor(id AggregateId, loaded []Event, result []Event) {
}
