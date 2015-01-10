package main

type MessageDispatcher struct {
	commandHandlers	map[Command]CommandHandler
}

func (md *MessageDispatcher) AddHandlerFor(c Command, h CommandHandler) {
}

func (md *MessageDispatcher) SendCommand(c Command) error {
	return nil
}

func main() {}
