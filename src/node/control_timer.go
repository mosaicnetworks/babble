package node

import (
	"math/rand"
	"time"
)

type timerFactory func(time.Duration) <-chan time.Time

// controlTimer controls the node's heartbeat.
type controlTimer struct {
	timerFactory timerFactory
	tickCh       chan struct{}      // sends a signal to listening process
	resetCh      chan time.Duration // receives instruction to reset the heartbeatTimer
	stopCh       chan struct{}      // receives instruction to stop the heartbeatTimer
	shutdownCh   chan struct{}      // receives instruction to exit Run loop
	isSet        bool
	isShutdown   bool
}

// newControlTimer is a controlTimer factory method.
func newControlTimer(timerFactory timerFactory) *controlTimer {
	return &controlTimer{
		timerFactory: timerFactory,
		tickCh:       make(chan struct{}),
		resetCh:      make(chan time.Duration),
		stopCh:       make(chan struct{}),
		shutdownCh:   make(chan struct{}),
	}
}

// newRandomcontrolTimer creates a new controlTimer that ticks at random
// intervals between min and 2*min, where min is a specified minimum time
// interval.
func newRandomControlTimer() *controlTimer {
	randomTimeout := func(min time.Duration) <-chan time.Time {
		if min == 0 {
			return nil
		}
		extra := (time.Duration(rand.Int63()) % min)
		return time.After(min + extra)
	}
	return newControlTimer(randomTimeout)
}

// run starts the Control Timer
func (c *controlTimer) run(init time.Duration) {

	setTimer := func(t time.Duration) <-chan time.Time {
		c.isSet = true
		return c.timerFactory(t)
	}

	timer := setTimer(init)
	for {
		select {
		case <-timer:
			c.tickCh <- struct{}{}
			c.isSet = false
		case t := <-c.resetCh:
			timer = setTimer(t)
		case <-c.stopCh:
			timer = nil
			c.isSet = false
		case <-c.shutdownCh:
			c.isSet = false
			return
		}
	}
}

// shutdown shuts down the controlTimer
func (c *controlTimer) shutdown() {
	if !c.isShutdown {
		close(c.shutdownCh)
		c.isShutdown = true
	}
	return
}
