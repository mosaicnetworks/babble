package node

import (
	"math/rand"
	"time"
)

type timerFactory func(time.Duration) <-chan time.Time

type ControlTimer struct {
	timerFactory timerFactory
	tickCh       chan struct{}      //sends a signal to listening process
	resetCh      chan time.Duration //receives instruction to reset the heartbeatTimer
	stopCh       chan struct{}      //receives instruction to stop the heartbeatTimer
	shutdownCh   chan struct{}      //receives instruction to exit Run loop
	set          bool
}

func NewControlTimer(timerFactory timerFactory) *ControlTimer {
	return &ControlTimer{
		timerFactory: timerFactory,
		tickCh:       make(chan struct{}),
		resetCh:      make(chan time.Duration),
		stopCh:       make(chan struct{}),
		shutdownCh:   make(chan struct{}),
	}
}

func NewRandomControlTimer() *ControlTimer {

	randomTimeout := func(min time.Duration) <-chan time.Time {
		if min == 0 {
			return nil
		}
		extra := (time.Duration(rand.Int63()) % min)
		return time.After(min + extra)
	}
	return NewControlTimer(randomTimeout)
}

func (c *ControlTimer) Run(init time.Duration) {

	setTimer := func(t time.Duration) <-chan time.Time {
		c.set = true
		return c.timerFactory(t)
	}

	timer := setTimer(init)
	for {
		select {
		case <-timer:
			c.tickCh <- struct{}{}
			c.set = false
		case t := <-c.resetCh:
			timer = setTimer(t)
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
