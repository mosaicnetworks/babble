package node

import "sync/atomic"

// NodeState captures the state of a Babble node: Babbling, CatchingUp or Shutdown
type NodeState uint32

const (
	// Babbling is the initial state of a Babble node.
	Babbling NodeState = iota

	CatchingUp

	Shutdown
)

func (s NodeState) String() string {
	switch s {
	case Babbling:
		return "Babbling"
	case CatchingUp:
		return "CatchingUp"
	case Shutdown:
		return "Shutdown"
	default:
		return "Unknown"
	}
}

type nodeState struct {
	state NodeState
}

func (b *nodeState) getState() NodeState {
	stateAddr := (*uint32)(&b.state)
	return NodeState(atomic.LoadUint32(stateAddr))
}

func (b *nodeState) setState(s NodeState) {
	stateAddr := (*uint32)(&b.state)
	atomic.StoreUint32(stateAddr, uint32(s))
}
