package node

import (
	"sync"
	"sync/atomic"
)

// State captures the state of a Babble node: Babbling, CatchingUp, Joining,
// Leaving, Suspended, or Shutdown
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
	//Suspended is initialised, but not gossipping
	Suspended
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
	case Suspended:
		return "Suspended"
	default:
		return "Unknown"
	}
}

// WGLIMIT is the maximum number of goroutines that can be launched through
// state.goFunc
const WGLIMIT = 20

type state struct {
	state   State
	wg      sync.WaitGroup
	wgCount int32
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
	tempWgCount := atomic.LoadInt32(&b.wgCount)
	if tempWgCount < WGLIMIT {
		b.wg.Add(1)
		atomic.AddInt32(&b.wgCount, 1)
		go func() {
			defer b.wg.Done()
			atomic.AddInt32(&b.wgCount, -1)
			f()
		}()
	}
}

func (b *state) waitRoutines() {
	b.wg.Wait()
}
