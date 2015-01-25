package cqrs

// An event is something that happened
// as a result of a command;
// for example, FaceSlapped.
type Event interface {
	Id() AggregateId
}

type EventListener interface {
	apply(e Event) error
}

// Persist and restore events.
type EventStorer interface {
	LoadEventsFor(AggregateId) ([]Event, error)
	SaveEventsFor(AggregateId, []Event, []Event)
}

type NullEventStorer struct {}

func (es *NullEventStorer) LoadEventsFor(id AggregateId) ([]Event, error) {
	return []Event{}, nil
}

func (es *NullEventStorer) SaveEventsFor(id AggregateId, loaded []Event, result []Event) {
}

type FileSystemEventStorer struct {
	rootdir	string
}

func (es *FileSystemEventStorer) LoadEventsFor(id AggregateId) ([]Event, error) {
	return []Event{}, nil
}

func (es *FileSystemEventStorer) SaveEventsFor(id AggregateId, loaded []Event, result []Event) {
}
