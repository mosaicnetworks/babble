package mobile

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/babbleio/babble/crypto"
)

type CommitHandler interface {
	OnCommit(*TxContext)
}

type ErrorHandler interface {
	OnError(*ErrorContext)
}

// NodeInfo object containing info describing the node that commited given transactions
type NodeInfo struct {
	ID int
}

// TxContext object containing commited blocks from other nodes
type TxContext struct {
	Info *NodeInfo
	Ball *Ball
	Data string
}

type ErrorContext struct {
	Error string
}

// Ball struct to hold ball position
type Ball struct {
	X               int32
	Y               int32
	Size            int32
	CircleBackColor int32 //the int32 representation of name of backColor ex. "BLUE"
	CircleForeColor int32 //the int32 representation of foreColor ex. "YELLOW"
}

type EventHandler struct {
	onMessage CommitHandler
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
func (ev *EventHandler) OnMessage(handler CommitHandler) {
	ev.onMessage = handler
}

// OnError set ErrorHandler on EventHandler
func (ev *EventHandler) OnError(handler ErrorHandler) {
	ev.onError = handler
}

func (ev *EventHandler) SetHandlers(err ErrorHandler, msg CommitHandler) {
	ev.onError = err
	ev.onMessage = msg
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

func (s *Subscription) error(error string) {
	var handler ErrorHandler
	if s.events != nil && s.events.onError != nil {
		handler = s.events.onError
	}

	ctx := ErrorContext{Error: error}
	handler.OnError(&ctx)
}

func (s *Subscription) commitTx(nodeID int, transactions [][]byte) {
	for _, tx := range transactions {
		ball := Ball{}
		size := binary.Size(ball)

		if len(tx) > 4 {
			if tx[0] == '*' && tx[1] == 'b' {

				nodeID = int(tx[2]) + (int(tx[3]) << 8)

				r := bytes.NewReader(tx[4:])

				if len(tx)-4 != size {
					s.error("Received tx incomplete size")
					continue
				}

				if err := binary.Read(r, binary.LittleEndian, &ball); err != nil {
					s.error(fmt.Sprintf("Fail to read commited tran. %s", err.Error()))
				}

				if s.events != nil && s.events.onMessage != nil {
					handler := s.events.onMessage

					ctx := TxContext{
						Info: &NodeInfo{ID: nodeID},
						Ball: &ball,
					}

					handler.OnCommit(&ctx)
				}
			}
		} else {
			s.error("Received tx of incompatible format")
		}
	}
}

func GetPrivPublKeys() string {
	pemDump, err := crypto.GeneratePemKey()
	if err != nil {
		fmt.Println("Error generating PemDump")
		os.Exit(2)
	}
	return pemDump.PublicKey + "=!@#@!=" + pemDump.PrivateKey
}
