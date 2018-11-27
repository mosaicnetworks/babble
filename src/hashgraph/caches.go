package hashgraph

import (
	"fmt"
	"sort"
	"strconv"

	cm "github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/peers"
)

type Key struct {
	x, y string
}

func (k Key) ToString() string {
	return fmt.Sprintf("{%s, %s}", k.x, k.y)
}

type TreKey struct {
	x, y, z string
}

func (k TreKey) ToString() string {
	return fmt.Sprintf("{%s, %s, %s}", k.x, k.y, k.z)
}

//------------------------------------------------------------------------------

type ParticipantEventsCache struct {
	participants *peers.PeerSet
	rim          *cm.RollingIndexMap
}

func NewParticipantEventsCache(size int, participants *peers.PeerSet) *ParticipantEventsCache {
	return &ParticipantEventsCache{
		participants: participants,
		rim:          cm.NewRollingIndexMap("ParticipantEvents", size, participants.IDs()),
	}
}

func (pec *ParticipantEventsCache) AddPeer(peer *peers.Peer) error {
	pec.participants = pec.participants.WithNewPeer(peer)
	return pec.rim.AddKey(peer.ID)
}

func (pec *ParticipantEventsCache) participantID(participant string) (uint32, error) {
	peer, ok := pec.participants.ByPubKey[participant]

	if !ok {
		return 0, cm.NewStoreErr("ParticipantEvents", cm.UnknownParticipant, participant)
	}

	return peer.ID, nil
}

//Get returns participant events with index > skip
func (pec *ParticipantEventsCache) Get(participant string, skipIndex int) ([]string, error) {
	id, err := pec.participantID(participant)
	if err != nil {
		return []string{}, err
	}

	pe, err := pec.rim.Get(id, skipIndex)
	if err != nil {
		return []string{}, err
	}

	res := make([]string, len(pe))
	for k := 0; k < len(pe); k++ {
		res[k] = pe[k].(string)
	}
	return res, nil
}

func (pec *ParticipantEventsCache) GetItem(participant string, index int) (string, error) {
	id, err := pec.participantID(participant)
	if err != nil {
		return "", err
	}

	item, err := pec.rim.GetItem(id, index)
	if err != nil {
		return "", err
	}
	return item.(string), nil
}

func (pec *ParticipantEventsCache) GetLast(participant string) (string, error) {
	id, err := pec.participantID(participant)
	if err != nil {
		return "", err
	}

	last, err := pec.rim.GetLast(id)
	if err != nil {
		return "", err
	}
	return last.(string), nil
}

func (pec *ParticipantEventsCache) Set(participant string, hash string, index int) error {
	id, err := pec.participantID(participant)
	if err != nil {
		return err
	}
	return pec.rim.Set(id, hash, index)
}

//returns [participant id] => lastKnownIndex
func (pec *ParticipantEventsCache) Known() map[uint32]int {
	return pec.rim.Known()
}

//------------------------------------------------------------------------------

type PeerSetCache struct {
	rounds   sort.IntSlice
	peerSets map[int]*peers.PeerSet
}

func NewPeerSetCache() *PeerSetCache {
	return &PeerSetCache{
		rounds:   sort.IntSlice{},
		peerSets: make(map[int]*peers.PeerSet),
	}
}

func (c *PeerSetCache) Set(round int, peerSet *peers.PeerSet) error {
	if _, ok := c.peerSets[round]; ok {
		return cm.NewStoreErr("PeerSetCache", cm.KeyAlreadyExists, strconv.Itoa(round))
	}
	c.peerSets[round] = peerSet
	c.rounds = append(c.rounds, round)
	c.rounds.Sort()
	return nil

}

func (c *PeerSetCache) Get(round int) (*peers.PeerSet, error) {
	//check if direclty in peerSets
	ps, ok := c.peerSets[round]
	if ok {
		return ps, nil
	}

	//situate round in sorted rounds
	if len(c.rounds) == 0 {
		return nil, cm.NewStoreErr("PeerSetCache", cm.KeyNotFound, strconv.Itoa(round))
	}

	if len(c.rounds) == 1 {
		if round < c.rounds[0] {
			return nil, cm.NewStoreErr("PeerSetCache", cm.KeyNotFound, strconv.Itoa(round))
		}
		return c.peerSets[c.rounds[0]], nil
	}

	for i := 0; i < len(c.rounds)-1; i++ {
		if round >= c.rounds[i] && round < c.rounds[i+1] {
			return c.peerSets[c.rounds[i]], nil
		}
	}

	//return last PeerSet
	return c.peerSets[c.rounds[len(c.rounds)-1]], nil
}

func (c *PeerSetCache) GetLast() (*peers.PeerSet, error) {
	if len(c.rounds) == 0 {
		return nil, cm.NewStoreErr("PeerSetCache", cm.NoPeerSet, "")
	}
	return c.peerSets[c.rounds[len(c.rounds)-1]], nil
}
