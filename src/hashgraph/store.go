package hashgraph

import "github.com/mosaicnetworks/babble/src/peers"

type Store interface {
	CacheSize() int
	GetPeerSet(int) (*peers.PeerSet, error)
	GetLastPeerSet() (*peers.PeerSet, error)
	SetPeerSet(int, *peers.PeerSet) error
	RepertoireByPubKey() map[string]*peers.Peer
	RepertoireByID() map[uint32]*peers.Peer
	RootsBySelfParent() map[string]*Root
	GetEvent(string) (*Event, error)
	SetEvent(*Event) error
	ParticipantEvents(string, int) ([]string, error)
	ParticipantEvent(string, int) (string, error)
	LastEventFrom(string) (string, bool, error)
	LastConsensusEventFrom(string) (string, bool, error)
	KnownEvents() map[uint32]int
	ConsensusEvents() []string
	ConsensusEventsCount() int
	AddConsensusEvent(*Event) error
	GetRound(int) (*RoundInfo, error)
	SetRound(int, *RoundInfo) error
	LastRound() int
	RoundWitnesses(int) []string
	RoundEvents(int) int
	GetRoot(string) (*Root, error)
	GetBlock(int) (*Block, error)
	SetBlock(*Block) error
	LastBlockIndex() int
	GetFrame(int) (*Frame, error)
	SetFrame(*Frame) error
	Reset(*Frame) error
	Close() error
	NeedBoostrap() bool // Was the store loaded from existing db
	StorePath() string
}
