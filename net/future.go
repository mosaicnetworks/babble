/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package net

import "time"

// Future is used to represent an action that may occur in the future.
type Future interface {
	// Error blocks until the future arrives and then
	// returns the error status of the future.
	// This may be called any number of times - all
	// calls will return the same value.
	// Note that it is not OK to call this method
	// twice concurrently on the same Future instance.
	Error() error
}

// deferError can be embedded to allow a future
// to provide an error in the future.
type deferError struct {
	err       error
	errCh     chan error
	responded bool
}

func (d *deferError) init() {
	d.errCh = make(chan error, 1)
}

func (d *deferError) Error() error {
	if d.err != nil {
		// Note that when we've received a nil error, this
		// won't trigger, but the channel is closed after
		// send so we'll still return nil below.
		return d.err
	}
	if d.errCh == nil {
		panic("waiting for response on nil channel")
	}
	d.err = <-d.errCh
	return d.err
}

func (d *deferError) respond(err error) {
	if d.errCh == nil {
		return
	}
	if d.responded {
		return
	}
	d.errCh <- err
	close(d.errCh)
	d.responded = true
}

// SyncFuture is used to return information about a pipelined Sync request.
type SyncFuture interface {
	Future

	// Start returns the time that the append request was started.
	// It is always OK to call this method.
	Start() time.Time

	// Request holds the parameters of the Sync call.
	// It is always OK to call this method.
	Request() *SyncRequest

	// Response holds the results of the Sync call.
	// This method must only be called after the Error
	// method returns, and will only be valid on success.
	Response() *SyncResponse
}

//SyncFuture is used for waiting on a pipelined sync RPC.
type syncFuture struct {
	deferError
	start time.Time
	args  *SyncRequest
	resp  *SyncResponse
}

func (s *syncFuture) Start() time.Time {
	return s.start
}

func (s *syncFuture) Request() *SyncRequest {
	return s.args
}

func (s *syncFuture) Response() *SyncResponse {
	return s.resp
}
