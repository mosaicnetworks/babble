package hashgraph

import "github.com/mosaicnetworks/babble/src/peers"

// Store is an interface for backend stores.
type Store interface {
	// CacheSize retrieves the cacheSize setting that determines the maximum
	// number of items that caches can contain.
	CacheSize() int
	// GetPeerSet returns the peer-set effective at a given round.
	GetPeerSet(round int) (*peers.PeerSet, error)
	// SetPeerSet sets the peer-set effective at a given round.
	SetPeerSet(round int, peers *peers.PeerSet) error
	// GetAllPeerSets returns the entire history of peer-sets.
	GetAllPeerSets() (map[int][]*peers.Peer, error)
	// FirstRound returns the index of the first round to which a participant
	// belonged.
	FirstRound(participantID uint32) (int, bool)
	// RepertoireByPubKey returns all the known peers by public key.
	RepertoireByPubKey() map[string]*peers.Peer
	// RepertoireByID returns all the known peers by ID.
	RepertoireByID() map[uint32]*peers.Peer
	// GetEvent returns an evnet by hash.
	GetEvent(hash string) (*Event, error)
	// SetEvent inserts an envent in the store.
	SetEvent(event *Event) error
	// ParticipantEvents returns all the sorted event hashes of a participant
	// starting at index skip+1.
	ParticipantEvents(participant string, skip int) ([]string, error)
	// ParticipantEvent returns a participant's event with a given index.
	ParticipantEvent(participant string, index int) (string, error)
	// LastEventFrom returns the last event of a participant.
	LastEventFrom(participant string) (string, error)
	// LastConsensusEventFrom returns the last consensus event produced by a
	// participant.
	LastConsensusEventFrom(string) (string, error)
	// KnownEvents returns the map of participant ID to last known index.
	KnownEvents() map[uint32]int
	// ConsensusEvents returns the hashes of consensus events.
	ConsensusEvents() []string
	// ConsensusEventsCount returns the number of consensus events.
	ConsensusEventsCount() int
	// AddConsensusEvent adds a consensus event.
	AddConsensusEvent(*Event) error
	// GetRound retrieves a round by index.
	GetRound(roundIndex int) (*RoundInfo, error)
	// SetRound stores a round.
	SetRound(roundIndex int, roundInfo *RoundInfo) error
	// LastRound returns the index of the last created round.
	LastRound() int
	// RoundWitnesses returns the hashes of a round's witnesses.
	RoundWitnesses(roundIndex int) []string
	// RoundEvents returns the number of events in a round.
	RoundEvents(roundIndex int) int
	// GetRoot returns a participant's root.
	GetRoot(participant string) (*Root, error)
	// GetBlock returns a block by index.
	GetBlock(int) (*Block, error)
	// SetBlock store a block.
	SetBlock(*Block) error
	// LastBlockIndex returns the last block index.
	LastBlockIndex() int
	// GetFrame retrieves the frame associated to a round received.
	GetFrame(roundReceived int) (*Frame, error)
	// SetFrame stores a frame.
	SetFrame(*Frame) error
	// Reset resets a store from a frame.
	Reset(*Frame) error
	// Close closes the underlying database.
	Close() error
	// StorePath returns the filepath of the underlying database.
	StorePath() string
}
