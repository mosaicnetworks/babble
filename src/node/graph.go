package node

import (
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
)

type Infos struct {
	ParticipantEvents map[string]map[string]*hg.Event
	Rounds            []*hg.RoundInfo
	Blocks            []*hg.Block
}

type Graph struct {
	*Node
}

func (g *Graph) GetParticipantEvents() (map[string]map[string]*hg.Event, error) {
	res := make(map[string]map[string]*hg.Event)

	store := g.Node.core.hg.Store
	repertoire := g.Node.core.hg.Store.RepertoireByPubKey()

	for _, p := range repertoire {
		root, err := store.GetRoot(p.PubKeyHex)
		if err != nil {
			return res, err
		}

		evs, err := store.ParticipantEvents(p.PubKeyHex, root.GetHead().Index)
		if err != nil {
			return res, err
		}

		res[p.PubKeyHex] = make(map[string]*hg.Event)

		res[p.PubKeyHex][root.Head] = hg.NewEvent(
			[][]byte{},
			[]hg.BlockSignature{},
			[]string{},
			[]byte{},
			-1,
		)

		for _, e := range evs {
			event, err := store.GetEvent(e)
			if err != nil {
				return res, err
			}

			hash := event.Hex()

			res[p.PubKeyHex][hash] = event
		}
	}

	return res, nil
}

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

func NewGraph(n *Node) *Graph {
	return &Graph{
		Node: n,
	}
}
