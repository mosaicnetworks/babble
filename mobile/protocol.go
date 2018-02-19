package mobile

// NodeInfo object containing info describing the node that commited given transactions
type NodeInfo struct {
	Id int
}

// Message object containing commited blocks from other nodes
type Message struct {
	Info *NodeInfo
	Data string
	// TODO: define needed properties
}

type ErrorContext struct {
	Error string
}

type MessageHandler interface {
	OnMessage(*Message)
}

type ErrorHandler interface {
	OnError(*ErrorContext)
}

type EventHandler struct {
	onMessage MessageHandler
	onError   ErrorHandler
}

type Config struct {
	HeartBeat int
	MaxPool   int
	CacheSize int
	SyncLimit int
}

type Subscription struct {
	events *EventHandler
}

// NewEventHandler initilizes new EventHandler
func NewEventHandler() *EventHandler {
	return &EventHandler{}
}

// OnMessage set MessageHandler on EventHandler
func (ev *EventHandler) OnMessage(handler MessageHandler) {
	ev.onMessage = handler
}

// OnError set ErrorHandler on EventHandler
func (ev *EventHandler) OnError(handler ErrorHandler) {
	ev.onError = handler
}

// DefaultConfig return Config with default options
func DefaultConfig() *Config {
	return &Config{
		HeartBeat: 1000,
		MaxPool:   2,
		CacheSize: 500,
		SyncLimit: 1000,
	}
}

func (s *Subscription) raiseError(error string) {
	var handler ErrorHandler
	if s.events != nil && s.events.onError != nil {
		handler = s.events.onError
	}

	ctx := ErrorContext{Error: error}
	handler.OnError(&ctx)
}

func (s *Subscription) message(msg string) {
	var handler MessageHandler
	if s.events != nil && s.events.onMessage != nil {
		handler = s.events.onMessage
	}

	ctx := Message{Data: msg, Info: &NodeInfo{Id: 1}}
	handler.OnMessage(&ctx)
}
