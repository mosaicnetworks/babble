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

import (
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"time"
)

// NewInmemAddr returns a new in-memory addr with
// a randomly generate UUID as the ID.
func NewInmemAddr() string {
	return generateUUID()
}

// generateUUID is used to generate a random UUID.
func generateUUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Errorf("failed to read random bytes: %v", err))
	}

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%12x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16])
}

// inmemPipeline is used to pipeline requests for the in-mem transport.
type inmemPipeline struct {
	trans    *InmemTransport
	peer     *InmemTransport
	peerAddr string

	doneCh       chan SyncFuture
	inprogressCh chan *inmemPipelineInflight

	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex
}

type inmemPipelineInflight struct {
	future *syncFuture
	respCh <-chan RPCResponse
}

// InmemTransport Implements the Transport interface, to allow Swirlds to be
// tested in-memory without going over a network.
type InmemTransport struct {
	sync.RWMutex
	consumerCh chan RPC
	localAddr  string
	peers      map[string]*InmemTransport
	pipelines  []*inmemPipeline
	timeout    time.Duration
}

// NewInmemTransport is used to initialize a new transport
// and generates a random local address if none is specified
func NewInmemTransport(addr string) (string, *InmemTransport) {
	if addr == "" {
		addr = NewInmemAddr()
	}
	trans := &InmemTransport{
		consumerCh: make(chan RPC, 16),
		localAddr:  addr,
		peers:      make(map[string]*InmemTransport),
		timeout:    50 * time.Millisecond,
	}
	return addr, trans
}

// Consumer implements the Transport interface.
func (i *InmemTransport) Consumer() <-chan RPC {
	return i.consumerCh
}

// LocalAddr implements the Transport interface.
func (i *InmemTransport) LocalAddr() string {
	return i.localAddr
}

// SyncPipeline returns an interface that can be used to pipeline
// Sync requests.
func (i *InmemTransport) SyncPipeline(target string) (SyncPipeline, error) {
	i.RLock()
	peer, ok := i.peers[target]
	i.RUnlock()
	if !ok {
		return nil, fmt.Errorf("failed to connect to peer: %v", target)
	}
	pipeline := newInmemPipeline(i, peer, target)
	i.Lock()
	i.pipelines = append(i.pipelines, pipeline)
	i.Unlock()
	return pipeline, nil
}

// Sync implements the Transport interface.
func (i *InmemTransport) Sync(target string, args *SyncRequest, resp *SyncResponse) error {
	rpcResp, err := i.makeRPC(target, args, nil, i.timeout)
	if err != nil {
		return err
	}

	// Copy the result back
	out := rpcResp.Response.(*SyncResponse)
	*resp = *out
	return nil
}

// RequestKnown implements the Transport interface.
func (i *InmemTransport) RequestKnown(target string, args *KnownRequest, resp *KnownResponse) error {
	rpcResp, err := i.makeRPC(target, args, nil, i.timeout)
	if err != nil {
		return err
	}

	// Copy the result back
	out := rpcResp.Response.(*KnownResponse)
	*resp = *out
	return nil
}

func (i *InmemTransport) makeRPC(target string, args interface{}, r io.Reader, timeout time.Duration) (rpcResp RPCResponse, err error) {
	i.RLock()
	peer, ok := i.peers[target]
	i.RUnlock()

	if !ok {
		err = fmt.Errorf("failed to connect to peer: %v", target)
		return
	}

	// Send the RPC over
	respCh := make(chan RPCResponse)
	peer.consumerCh <- RPC{
		Command:  args,
		Reader:   r,
		RespChan: respCh,
	}

	// Wait for a response
	select {
	case rpcResp = <-respCh:
		if rpcResp.Error != nil {
			err = rpcResp.Error
		}
	case <-time.After(timeout):
		err = fmt.Errorf("command timed out")
	}
	return
}

// Connect is used to connect this transport to another transport for
// a given peer name. This allows for local routing.
func (i *InmemTransport) Connect(peer string, t Transport) {
	trans := t.(*InmemTransport)
	i.Lock()
	defer i.Unlock()
	i.peers[peer] = trans
}

// Disconnect is used to remove the ability to route to a given peer.
func (i *InmemTransport) Disconnect(peer string) {
	i.Lock()
	defer i.Unlock()
	delete(i.peers, peer)

	// Disconnect any pipelines
	n := len(i.pipelines)
	for idx := 0; idx < n; idx++ {
		if i.pipelines[idx].peerAddr == peer {
			i.pipelines[idx].Close()
			i.pipelines[idx], i.pipelines[n-1] = i.pipelines[n-1], nil
			idx--
			n--
		}
	}
	i.pipelines = i.pipelines[:n]
}

// DisconnectAll is used to remove all routes to peers.
func (i *InmemTransport) DisconnectAll() {
	i.Lock()
	defer i.Unlock()
	i.peers = make(map[string]*InmemTransport)

	// Handle pipelines
	for _, pipeline := range i.pipelines {
		pipeline.Close()
	}
	i.pipelines = nil
}

// Close is used to permanently disable the transport
func (i *InmemTransport) Close() error {
	i.DisconnectAll()
	return nil
}

func newInmemPipeline(trans *InmemTransport, peer *InmemTransport, addr string) *inmemPipeline {
	i := &inmemPipeline{
		trans:        trans,
		peer:         peer,
		peerAddr:     addr,
		doneCh:       make(chan SyncFuture, 16),
		inprogressCh: make(chan *inmemPipelineInflight, 16),
		shutdownCh:   make(chan struct{}),
	}
	go i.decodeResponses()
	return i
}

func (i *inmemPipeline) decodeResponses() {
	timeout := i.trans.timeout
	for {
		select {
		case inp := <-i.inprogressCh:
			var timeoutCh <-chan time.Time
			if timeout > 0 {
				timeoutCh = time.After(timeout)
			}

			select {
			case rpcResp := <-inp.respCh:
				// Copy the result back
				*inp.future.resp = *rpcResp.Response.(*SyncResponse)
				inp.future.respond(rpcResp.Error)

				select {
				case i.doneCh <- inp.future:
				case <-i.shutdownCh:
					return
				}

			case <-timeoutCh:
				inp.future.respond(fmt.Errorf("command timed out"))
				select {
				case i.doneCh <- inp.future:
				case <-i.shutdownCh:
					return
				}

			case <-i.shutdownCh:
				return
			}
		case <-i.shutdownCh:
			return
		}
	}
}

func (i *inmemPipeline) Sync(args *SyncRequest, resp *SyncResponse) (SyncFuture, error) {
	// Create a new future
	future := &syncFuture{
		start: time.Now(),
		args:  args,
		resp:  resp,
	}
	future.init()

	// Handle a timeout
	var timeout <-chan time.Time
	if i.trans.timeout > 0 {
		timeout = time.After(i.trans.timeout)
	}

	// Send the RPC over
	respCh := make(chan RPCResponse, 1)
	rpc := RPC{
		Command:  args,
		RespChan: respCh,
	}
	select {
	case i.peer.consumerCh <- rpc:
	case <-timeout:
		return nil, fmt.Errorf("command enqueue timeout")
	case <-i.shutdownCh:
		return nil, ErrPipelineShutdown
	}

	// Send to be decoded
	select {
	case i.inprogressCh <- &inmemPipelineInflight{future, respCh}:
		return future, nil
	case <-i.shutdownCh:
		return nil, ErrPipelineShutdown
	}
}

func (i *inmemPipeline) Consumer() <-chan SyncFuture {
	return i.doneCh
}

func (i *inmemPipeline) Close() error {
	i.shutdownLock.Lock()
	defer i.shutdownLock.Unlock()
	if i.shutdown {
		return nil
	}

	i.shutdown = true
	close(i.shutdownCh)
	return nil
}
