package node

import (
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
)

// Infos is the object used by Graph to collect information about a Hashgraph.
type Infos struct {
	ParticipantEvents map[string]map[string]*hg.Event
	Rounds            []*hg.RoundInfo
	Blocks            []*hg.Block
}

// Graph is a struct containing a node which is is used to collect information
// about the underlying Hashgraph in view of producing a visual representation
// of the Hashgraph.
type Graph struct {
	*Node
}

// NewGraph instantiates a Graph from a Node.
func NewGraph(n *Node) *Graph {
	return &Graph{
		Node: n,
	}
}

// GetParticipantEvents returns all the Events for all the participants that
// have been members of the group.
func (g *Graph) GetParticipantEvents() (map[string]map[string]*hg.Event, error) {
	res := make(map[string]map[string]*hg.Event)

	store := g.Node.core.hg.Store
	repertoire := g.Node.core.hg.Store.RepertoireByPubKey()

	for _, p := range repertoire {
		root, err := store.GetRoot(p.PubKeyString())
		if err != nil {
			return res, err
		}

		start := -1
		if l := len(root.Events); l > 0 {
			start = root.Events[l-1].Core.Index()
		}

		evs, err := store.ParticipantEvents(p.PubKeyString(), start)
		if err != nil {
			return res, err
		}

		res[p.PubKeyString()] = make(map[string]*hg.Event)

		for _, e := range evs {
			event, err := store.GetEvent(e)
			if err != nil {
				return res, err
			}

			hash := event.Hex()

			res[p.PubKeyString()][hash] = event
		}
	}

	return res, nil
}

// GetRounds returns all the recorded Hashgraph rounds.
func (g *Graph) GetRounds() []*hg.RoundInfo {
	res := []*hg.RoundInfo{}

	round := 0

	store := g.Node.core.hg.Store

	for round <= store.LastRound() {
		r, err := store.GetRound(round)

		if err != nil {
			break
		}

		res = append(res, r)

		round++
	}

	return res
}

// GetBlocks returns all the recorded blocks.
func (g *Graph) GetBlocks() []*hg.Block {
	res := []*hg.Block{}

	blockIdx := 0

	store := g.Node.core.hg.Store

	for blockIdx <= store.LastBlockIndex() {
		r, err := store.GetBlock(blockIdx)

		if err != nil {
			break
		}

		res = append(res, r)

		blockIdx++
	}

	return res
}

// GetInfos returns an Infos struct representing the entire Hashgraph.
func (g *Graph) GetInfos() (Infos, error) {
	participantEvents, err := g.GetParticipantEvents()
	if err != nil {
		return Infos{}, err
	}

	return Infos{
		ParticipantEvents: participantEvents,
		Rounds:            g.GetRounds(),
		Blocks:            g.GetBlocks(),
	}, nil
}
