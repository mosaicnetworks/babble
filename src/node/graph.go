package node

import (
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
)

type Infos struct {
	ParticipantEvents map[string]map[string]hg.Event
	Rounds            []hg.RoundInfo
	Blocks            []hg.Block
}

type Graph struct {
	*Node
}

func (g *Graph) GetParticipantEvents() map[string]map[string]hg.Event {
	res := make(map[string]map[string]hg.Event)

	store := g.Node.core.hg.Store
	peers := g.Node.core.hg.Participants

	for _, p := range peers.ByPubKey {
		root, err := store.GetRoot(p.PubKeyHex)

		if err != nil {
			panic(err)
		}

		evs, err := store.ParticipantEvents(p.PubKeyHex, root.SelfParent.Index)

		if err != nil {
			panic(err)
		}

		res[p.PubKeyHex] = make(map[string]hg.Event)

		res[p.PubKeyHex][root.SelfParent.Hash] = hg.NewEvent(
			[][]byte{},
			[]*hg.InternalTransaction{},
			[]hg.BlockSignature{},
			[]string{},
			[]byte{},
			-1,
		)

		for _, e := range evs {
			event, err := store.GetEvent(e)

			if err != nil {
				panic(err)
			}

			hash := event.Hex()

			res[p.PubKeyHex][hash] = event
		}
	}

	return res
}

func (g *Graph) GetRounds() []hg.RoundInfo {
	res := []hg.RoundInfo{}

	round := 0

	store := g.Node.core.hg.Store

	for round <= store.LastRound() {
		r, err := store.GetRound(round)

		if err != nil || !r.IsQueued() {
			break
		}

		res = append(res, r)

		round++
	}

	return res
}

func (g *Graph) GetBlocks() []hg.Block {
	res := []hg.Block{}

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

func (g *Graph) GetInfos() Infos {
	return Infos{
		ParticipantEvents: g.GetParticipantEvents(),
		Rounds:            g.GetRounds(),
		Blocks:            g.GetBlocks(),
	}
}

func NewGraph(n *Node) *Graph {
	return &Graph{
		Node: n,
	}
}
