package hashgraph

import "errors"

var (
	ErrKeyNotFound = errors.New("not found")
	ErrTooLate     = errors.New("too late")
)

type Store interface {
	CacheSize() int
	GetEvent(string) (Event, error)
	SetEvent(Event) error
	ParticipantEvents(string, int) ([]string, error)
	ParticipantEvent(string, int) (string, error)
	LastFrom(string) (string, error)
	Known() map[int]int
	ConsensusEvents() []string
	ConsensusEventsCount() int
	AddConsensusEvent(string) error
	GetRound(int) (RoundInfo, error)
	SetRound(int, RoundInfo) error
	Rounds() int
	RoundWitnesses(int) []string
	RoundEvents(int) int
}
