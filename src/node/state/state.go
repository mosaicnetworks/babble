package state

import (
	"sync"
	"sync/atomic"
)

// State captures the state of a Babble node: Babbling, CatchingUp, Joining,
// Leaving, Suspended, or Shutdown
type State uint32

const (
	// Babbling is the state in which a node gossips regularly with other nodes
	// as part of the hashgraph consensus algorithm, and responds to other
	// requests.
	Babbling State = iota

	// CatchingUp is the state in which a node attempts to fast-forward to a
	// future point in the hashgraph as part of the FastSync protocol.
	CatchingUp

	// Joining is the state in which a node attempts to join a Babble group by
	// submitting a join request.
	Joining

	// Leaving is the state in which a node attempts to politely leave a Babble
	// group by submitting a leave request.
	Leaving

	// Shutdown is the state in which a node stops responding to external events
	// and closes its transport.
	Shutdown

	// Suspended is the state in which a node passively participates in the
	// gossip protocol but does not process any new events or transactions.
	Suspended
)

// WGLIMIT is the maximum number of goroutines that can be launched through
// state.GoFunc
const WGLIMIT = 20

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

// Manager wraps a State with get and set methods. It is also used to limit the
// number of goroutines launched by the node, and to wait for all of them to
// complete.
type Manager struct {
	state   State
	wg      sync.WaitGroup
	wgCount int32
}

// GetState returns the current state.
func (b *Manager) GetState() State {
	stateAddr := (*uint32)(&b.state)
	return State(atomic.LoadUint32(stateAddr))
}

// SetState sets the state.
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
