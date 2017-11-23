package node

import (
	"math/rand"
	"time"
)

type timerFactory func() <-chan time.Time

type ControlTimer struct {
	timerFactory timerFactory
	tickCh       chan struct{} //sends a signal to listening process
	resetCh      chan struct{} //receives instruction to reset the heartbeatTimer
	stopCh       chan struct{} //receives instruction to stop the heartbeatTimer
	shutdownCh   chan struct{} //receives instruction to exit Run loop
	set          bool
}

func NewControlTimer(timerFactory timerFactory) *ControlTimer {
	return &ControlTimer{
		timerFactory: timerFactory,
		tickCh:       make(chan struct{}),
		resetCh:      make(chan struct{}),
		stopCh:       make(chan struct{}),
		shutdownCh:   make(chan struct{}),
	}
}

func NewRandomControlTimer(base time.Duration) *ControlTimer {

	randomTimeout := func() <-chan time.Time {
		minVal := base
		if minVal == 0 {
			return nil
		}
		extra := (time.Duration(rand.Int63()) % minVal)
		return time.After(minVal + extra)
	}
	return NewControlTimer(randomTimeout)
}

func (c *ControlTimer) Run() {

	setTimer := func() <-chan time.Time {
		c.set = true
		return c.timerFactory()
	}

	timer := setTimer()
	for {
		select {
		case <-timer:
			c.tickCh <- struct{}{}
			c.set = false
		case <-c.resetCh:
			timer = setTimer()
		case <-c.stopCh:
			timer = nil
			c.set = false
		case <-c.shutdownCh:
			c.set = false
			return
		}
	}
}

func (c *ControlTimer) Shutdown() {
	close(c.shutdownCh)
}
