package state

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

// String returns the string representation of a State
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

// Manager wraps a State with get and set methods. It is also used to limit the
// number of goroutines launched by the node, and to wait for all of them to
// complete.
type Manager struct {
	state   State
	wg      sync.WaitGroup
	wgCount int32
}

// GetState returns the current state
func (b *Manager) GetState() State {
	stateAddr := (*uint32)(&b.state)
	return State(atomic.LoadUint32(stateAddr))
}

// SetState sets the state
func (b *Manager) SetState(s State) {
	stateAddr := (*uint32)(&b.state)
	atomic.StoreUint32(stateAddr, uint32(s))
}

// GoFunc launches a goroutine for a given function, if there are currently
// less than WGLIMIT running. It increments the waitgroup.
func (b *Manager) GoFunc(f func()) {
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

// WaitRoutines waits for all the goroutines in the waitgroup.
func (b *Manager) WaitRoutines() {
	b.wg.Wait()
}
