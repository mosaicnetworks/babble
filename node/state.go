package node

import (
	"sync"
	"sync/atomic"
)

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
	state    NodeState
	starting int32
	wg       sync.WaitGroup
}

func setRecording(shoudRecord bool) {

}

func (b *nodeState) getState() NodeState {
	stateAddr := (*uint32)(&b.state)
	return NodeState(atomic.LoadUint32(stateAddr))
}

func (b *nodeState) setState(s NodeState) {
	stateAddr := (*uint32)(&b.state)
	atomic.StoreUint32(stateAddr, uint32(s))
}

func (b *nodeState) isStarting() bool {
	return b.starting > 0
}

func (b *nodeState) setStarting(starting bool) {
	if starting {
		atomic.CompareAndSwapInt32(&b.starting, 0, 1)
	} else {
		atomic.CompareAndSwapInt32(&b.starting, 1, 0)
	}
}

// Start a goroutine and add it to waitgroup
func (b *nodeState) goFunc(f func()) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		f()
	}()
}

func (b *nodeState) waitRoutines() {
	b.wg.Wait()
}
