package node

import (
	"sync"
	"sync/atomic"
)

// State captures the state of a Babble node: Babbling, CatchingUp, Joining,
// or Shutdown
type State uint32

const (
	//Babbling is the initial state of a Babble node.
	Babbling State = iota
	//CatchingUp implements Fast Sync
	CatchingUp
	//Joining is joining
	Joining
	//Leaving is leaving
	Leaving
	//Shutdown is shutdown
	Shutdown
)

// String ...
func (s State) String() string {
	switch s {
	case Babbling:
		return "Babbling"
	case CatchingUp:
		return "CatchingUp"
	case Joining:
		return "Joining"
	case Leaving:
		return "Leaving"
	case Shutdown:
		return "Shutdown"
	default:
		return "Unknown"
	}
}

type state struct {
	state State
	wg    sync.WaitGroup
}

func (b *state) getState() State {
	stateAddr := (*uint32)(&b.state)
	return State(atomic.LoadUint32(stateAddr))
}

func (b *state) setState(s State) {
	stateAddr := (*uint32)(&b.state)
	atomic.StoreUint32(stateAddr, uint32(s))
}

// Start a goroutine and add it to waitgroup
func (b *state) goFunc(f func()) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		f()
	}()
}

func (b *state) waitRoutines() {
	b.wg.Wait()
}
