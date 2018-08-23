package hashgraph

type Store interface {
	CacheSize() int
	Participants() (map[string]int, error)
	RootsBySelfParent() (map[string]Root, error)
	GetEvent(string) (Event, error)
	SetEvent(Event) error
	ParticipantEvents(string, int) ([]string, error)
	ParticipantEvent(string, int) (string, error)
	LastEventFrom(string) (string, bool, error)
	LastConsensusEventFrom(string) (string, bool, error)
	KnownEvents() map[int]int
	ConsensusEvents() []string
	ConsensusEventsCount() int
	AddConsensusEvent(Event) error
	GetRound(int) (RoundInfo, error)
	SetRound(int, RoundInfo) error
	LastRound() int
	RoundWitnesses(int) []string
	RoundEvents(int) int
	GetRoot(string) (Root, error)
	GetBlock(int) (Block, error)
	SetBlock(Block) error
	LastBlockIndex() int
	GetFrame(int) (Frame, error)
	SetFrame(Frame) error
	Reset(map[string]Root) error
	Close() error
}
