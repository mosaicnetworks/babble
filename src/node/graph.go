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

func (g *Graph) GetParticipantEvents() map[string]map[string]*hg.Event {
	res := make(map[string]map[string]*hg.Event)

	store := g.Node.core.hg.Store
	repertoire := g.Node.core.hg.Store.RepertoireByPubKey()
	known := store.KnownEvents()

	for _, p := range repertoire {
		root, err := store.GetRoot(p.PubKeyHex)

		if err != nil {
			panic(err)
		}

		skip := known[p.ID] - 30
		if skip < 0 {
			skip = -1
		}

		evs, err := store.ParticipantEvents(p.PubKeyHex, skip)

		if err != nil {
			panic(err)
		}

		res[p.PubKeyHex] = make(map[string]*hg.Event)

		res[p.PubKeyHex][root.SelfParent.Hash] = hg.NewEvent(
			[][]byte{},
			[]hg.InternalTransaction{},
			[]hg.BlockSignature{},
			[]string{},
			[]byte{},
			-1,
		)

		isFirst := true

		for _, e := range evs {
			event, err := store.GetEvent(e)
			ev := *event

			if err != nil {
				panic(err)
			}

			hash := event.Hex()

			if isFirst {
				isFirst = false
				ev.Body.Parents = []string{}
			}

			res[p.PubKeyHex][hash] = &ev
		}
	}

	return res
}

func (g *Graph) GetRounds() []*hg.RoundInfo {
	res := []*hg.RoundInfo{}

	store := g.Node.core.hg.Store

	round := store.LastRound() - 20

	if round < 0 {
		round = 0
	}

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

	store := g.Node.core.hg.Store

	blockIdx := store.LastBlockIndex() - 10

	if blockIdx < 0 {
		blockIdx = 0
	}

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
