package node

import hg "github.com/mosaicnetworks/babble/src/hashgraph"

type Infos struct {
	ParticipantEvents map[string]map[string]hg.Event
	Rounds            []hg.RoundInfo
	Blocks            []hg.Block
}

type Graph struct {
	Node *Node
}

func (g *Graph) GetParticipantEvents() map[string]map[string]hg.Event {
	res := make(map[string]map[string]hg.Event)

	for _, p := range g.Node.core.participants.ByPubKey {
		root, _ := g.Node.core.hg.Store.GetRoot(p.PubKeyHex)

		evs, _ := g.Node.core.hg.Store.ParticipantEvents(p.PubKeyHex, -1)

		res[p.PubKeyHex] = make(map[string]hg.Event)

		res[p.PubKeyHex][root.SelfParent.Hash] = hg.NewEvent([][]byte{}, []hg.BlockSignature{}, []string{}, []byte{}, -1)

		for _, e := range evs {
			event, _ := g.Node.core.GetEvent(e)

			hash := event.Hex()

			res[p.PubKeyHex][hash] = event
		}
	}

	return res
}

func (g *Graph) GetRounds() []hg.RoundInfo {
	res := []hg.RoundInfo{}

	round := 0

	for round <= g.Node.core.hg.Store.LastRound() {
		r, err := g.Node.core.hg.Store.GetRound(round)

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

	for blockIdx <= g.Node.core.hg.Store.LastBlockIndex() {
		r, err := g.Node.core.hg.Store.GetBlock(blockIdx)

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
