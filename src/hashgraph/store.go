package hashgraph

import "github.com/mosaicnetworks/babble/src/peers"

// Store ...
type Store interface {
	CacheSize() int
	GetPeerSet(int) (*peers.PeerSet, error)
	SetPeerSet(int, *peers.PeerSet) error
	GetAllPeerSets() (map[int][]*peers.Peer, error)
	FirstRound(uint32) (int, bool)
	RepertoireByPubKey() map[string]*peers.Peer
	RepertoireByID() map[uint32]*peers.Peer
	GetEvent(string) (*Event, error)
	SetEvent(*Event) error
	ParticipantEvents(string, int) ([]string, error)
	ParticipantEvent(string, int) (string, error)
	LastEventFrom(string) (string, error)
	LastConsensusEventFrom(string) (string, error)
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
	StorePath() string
}
