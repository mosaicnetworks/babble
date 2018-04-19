package hashgraph

type Store interface {
	CacheSize() int
	Participants() (map[string]int, error)
	GetEvent(string) (Event, error)
	SetEvent(Event) error
	ParticipantEvents(string, int) ([]string, error)
	ParticipantEvent(string, int) (string, error)
	LastEventFrom(string) (string, bool, error)
	KnownEvents() map[int]int
	ConsensusEvents() []string
	ConsensusEventsCount() int
	AddConsensusEvent(string) error
	GetRound(int) (RoundInfo, error)
	SetRound(int, RoundInfo) error
	LastRound() int
	RoundWitnesses(int) []string
	RoundEvents(int) int
	GetRoot(string) (Root, error)
	GetBlock(int) (Block, error)
	SetBlock(Block) error
	Reset(map[string]Root) error
	Close() error
}
