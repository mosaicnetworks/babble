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

// InmemTransport Implements the Transport interface, to allow babble to be
// tested in-memory without going over a network.
type InmemTransport struct {
	sync.RWMutex
	consumerCh chan RPC
	localAddr  string
	peers      map[string]*InmemTransport
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

// AdvertiseAddr implements the Transport interface.
func (i *InmemTransport) AdvertiseAddr() string {
	return i.localAddr
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

// EagerSync implements the Transport interface.
func (i *InmemTransport) EagerSync(target string, args *EagerSyncRequest, resp *EagerSyncResponse) error {
	rpcResp, err := i.makeRPC(target, args, nil, i.timeout)
	if err != nil {
		return err
	}

	// Copy the result back
	out := rpcResp.Response.(*EagerSyncResponse)
	*resp = *out
	return nil
}

// FastForward implements the Transport interface.
func (i *InmemTransport) FastForward(target string, args *FastForwardRequest, resp *FastForwardResponse) error {
	rpcResp, err := i.makeRPC(target, args, nil, i.timeout)
	if err != nil {
		return err
	}

	// Copy the result back
	out := rpcResp.Response.(*FastForwardResponse)
	*resp = *out
	return nil
}

// Join implements the Transport interface
func (i *InmemTransport) Join(target string, args *JoinRequest, resp *JoinResponse) error {
	rpcResp, err := i.makeRPC(target, args, nil, i.timeout)
	if err != nil {
		return err
	}

	// Copy the result back
	out := rpcResp.Response.(*JoinResponse)
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
}

// DisconnectAll is used to remove all routes to peers.
func (i *InmemTransport) DisconnectAll() {
	i.Lock()
	defer i.Unlock()
	i.peers = make(map[string]*InmemTransport)
}

// Close is used to permanently disable the transport
func (i *InmemTransport) Close() error {
	i.DisconnectAll()
	return nil
}

// Listen is an empty function as there is no need to defer
// initialisation of the InMem service
func (i *InmemTransport) Listen() {
}
