package net

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

/*******************************************************************************
MOST OF THIS IS TAKEN FROM HASHICORP RAFT
*******************************************************************************/

const (
	rpcJoin uint8 = iota
	rpcSync
	rpcEagerSync
	rpcFastForward
)

const (
	// we need this high buffer size for compatibility with WebRTC
	bufSize = math.MaxUint16
)

var (
	// ErrTransportShutdown is returned when operations on a transport are
	// invoked after it's been terminated.
	ErrTransportShutdown = errors.New("transport shutdown")
)

/*
NetworkTransport provides a network based transport that can be
used to communicate with babble on remote machines. It requires
an underlying stream layer to provide a stream abstraction, which can
be simple TCP, TLS, etc.

This transport is very simple and lightweight. Each RPC request is
framed by sending a byte that indicates the message type, followed
by the json encoded request.

The response is an error string followed by the response object,
both are encoded using msgpack
*/
type NetworkTransport struct {
	logger *logrus.Entry

	connPool     map[string][]*netConn
	connPoolLock sync.Mutex
	maxPool      int

	consumeCh chan RPC

	shutdown     bool
	shutdownCh   chan struct{}
	shutdownLock sync.Mutex

	stream StreamLayer

	timeout     time.Duration
	joinTimeout time.Duration
}

type netConn struct {
	target string
	conn   net.Conn
	r      *bufio.Reader
	w      *bufio.Writer
	dec    *json.Decoder
	enc    *json.Encoder
}

// Release closes the underlying connection
func (n *netConn) Release() error {
	return n.conn.Close()
}

// NewNetworkTransport creates a new network transport with the given dialer
// and listener. The maxPool controls how many connections we will pool (per
// target). The timeout is used to apply I/O deadlines.
func NewNetworkTransport(
	stream StreamLayer,
	maxPool int,
	timeout time.Duration,
	joinTimeout time.Duration,
	logger *logrus.Entry,
) *NetworkTransport {

	if logger == nil {
		log := logrus.New()
		log.Level = logrus.DebugLevel
		logger = logrus.NewEntry(log)
	}

	trans := &NetworkTransport{
		connPool:    make(map[string][]*netConn),
		consumeCh:   make(chan RPC),
		logger:      logger,
		maxPool:     maxPool,
		shutdownCh:  make(chan struct{}),
		stream:      stream,
		timeout:     timeout,
		joinTimeout: joinTimeout,
	}

	return trans
}

// Close is used to stop the network transport.
func (n *NetworkTransport) Close() error {
	n.shutdownLock.Lock()
	defer n.shutdownLock.Unlock()

	if !n.shutdown {
		close(n.shutdownCh)
		n.stream.Close()

		n.shutdown = true
	}
	return nil
}

// Consumer implements the Transport interface.
func (n *NetworkTransport) Consumer() <-chan RPC {
	return n.consumeCh
}

// LocalAddr implements the Transport interface.
func (n *NetworkTransport) LocalAddr() string {
	addr := n.stream.Addr()

	if addr != nil {
		return addr.String()
	}

	return ""
}

// AdvertiseAddr implements the Transport interface.
func (n *NetworkTransport) AdvertiseAddr() string {
	return n.stream.AdvertiseAddr()
}

// IsShutdown is used to check if the transport is shutdown.
func (n *NetworkTransport) IsShutdown() bool {
	select {
	case <-n.shutdownCh:
		return true
	default:
		return false
	}
}

// getPooledConn is used to grab a pooled connection.
func (n *NetworkTransport) getPooledConn(target string) *netConn {
	n.connPoolLock.Lock()
	defer n.connPoolLock.Unlock()

	conns, ok := n.connPool[target]
	if !ok || len(conns) == 0 {
		return nil
	}

	var conn *netConn
	num := len(conns)
	conn, conns[num-1] = conns[num-1], nil
	n.connPool[target] = conns[:num-1]
	return conn
}

// getConn is used to get a connection from the pool.
func (n *NetworkTransport) getConn(target string, timeout time.Duration) (*netConn, error) {
	// Check for a pooled conn
	if conn := n.getPooledConn(target); conn != nil {
		return conn, nil
	}

	// Dial a new connection
	conn, err := n.stream.Dial(target, timeout)
	if err != nil {
		return nil, err
	}

	// Wrap the conn
	netConn := &netConn{
		target: target,
		conn:   conn,
		r:      bufio.NewReaderSize(conn, bufSize),
		w:      bufio.NewWriterSize(conn, bufSize),
	}
	// Setup encoder/decoders
	netConn.dec = json.NewDecoder(netConn.r)
	netConn.enc = json.NewEncoder(netConn.w)

	// Done
	return netConn, nil
}

// returnConn returns a connection back to the pool.
func (n *NetworkTransport) returnConn(conn *netConn) {
	n.connPoolLock.Lock()
	defer n.connPoolLock.Unlock()

	key := conn.target
	conns, _ := n.connPool[key]

	if !n.IsShutdown() && len(conns) < n.maxPool {
		n.connPool[key] = append(conns, conn)
	} else {
		conn.Release()
	}
}

// Sync implements the Transport interface.
func (n *NetworkTransport) Sync(target string, args *SyncRequest, resp *SyncResponse) error {
	return n.genericRPC(target, rpcSync, n.timeout, args, resp)
}

// EagerSync implements the Transport interface.
func (n *NetworkTransport) EagerSync(target string, args *EagerSyncRequest, resp *EagerSyncResponse) error {
	return n.genericRPC(target, rpcEagerSync, n.timeout, args, resp)
}

// FastForward implements the Transport interface.
func (n *NetworkTransport) FastForward(target string, args *FastForwardRequest, resp *FastForwardResponse) error {
	return n.genericRPC(target, rpcFastForward, n.timeout, args, resp)
}

// Join implements the Transport interface.
func (n *NetworkTransport) Join(target string, args *JoinRequest, resp *JoinResponse) error {
	return n.genericRPC(target, rpcJoin, n.joinTimeout, args, resp)
}

// genericRPC handles a simple request/response RPC.
func (n *NetworkTransport) genericRPC(target string, rpcType uint8, timeout time.Duration, args interface{}, resp interface{}) error {
	// Get a conn
	conn, err := n.getConn(target, timeout)
	if err != nil {
		return err
	}

	// Set a deadline
	if timeout > 0 {
		conn.conn.SetDeadline(time.Now().Add(timeout))
	}

	// Send the RPC
	if err = sendRPC(conn, rpcType, args); err != nil {
		return err
	}

	// Decode the response
	canReturn, err := decodeResponse(conn, resp)
	if canReturn {
		n.returnConn(conn)
	}

	return err
}

// sendRPC is used to encode and send the RPC.
func sendRPC(conn *netConn, rpcType uint8, args interface{}) error {
	// Write the request type
	if err := conn.w.WriteByte(rpcType); err != nil {
		conn.Release()
		return err
	}

	// Send the request
	if err := conn.enc.Encode(args); err != nil {
		conn.Release()
		return err
	}

	// Flush
	if err := conn.w.Flush(); err != nil {
		conn.Release()
		return err
	}
	return nil
}

// decodeResponse is used to decode an RPC response and reports whether
// the connection can be reused.
func decodeResponse(conn *netConn, resp interface{}) (bool, error) {
	// Decode the error if any
	var rpcError string
	if err := conn.dec.Decode(&rpcError); err != nil {
		conn.Release()
		return false, err
	}

	// Decode the response
	if err := conn.dec.Decode(resp); err != nil {
		conn.Release()
		return false, err
	}

	// Format an error if any
	if rpcError != "" {
		return true, fmt.Errorf(rpcError)
	}
	return true, nil
}

// Listen opens the stream and handles incoming connections.
func (n *NetworkTransport) Listen() {
	for {
		// Accept incoming connections
		conn, err := n.stream.Accept()
		if err != nil {
			if n.IsShutdown() {
				return
			}
			n.logger.WithField("error", err).Error("Failed to accept connection")
			continue
		}
		n.logger.WithFields(logrus.Fields{
			"node": conn.LocalAddr(),
			"from": conn.RemoteAddr(),
		}).Debug("accepted connection")

		// Handle the connection in dedicated routine
		go n.handleConn(conn)
	}
}

// handleConn is used to handle an inbound connection for its lifespan.
func (n *NetworkTransport) handleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReaderSize(conn, bufSize)
	w := bufio.NewWriterSize(conn, bufSize)
	dec := json.NewDecoder(r)
	enc := json.NewEncoder(w)

	for {
		if err := n.handleCommand(r, dec, enc); err != nil {

			if err == ErrTransportShutdown {
				n.logger.WithField("error", err).Warn("Failed to decode incoming command")
			} else {
				if err != io.EOF {
					n.logger.WithField("error", err).Error("Failed to decode incoming command")
				}
			}
			return
		}
		if err := w.Flush(); err != nil {
			n.logger.WithField("error", err).Error("Failed to flush response")
			return
		}
	}
}

// handleCommand is used to decode and dispatch a single command.
func (n *NetworkTransport) handleCommand(r *bufio.Reader, dec *json.Decoder, enc *json.Encoder) error {
	// Get the rpc type
	rpcType, err := r.ReadByte()
	if err != nil {
		return err
	}

	// Create the RPC object
	respCh := make(chan RPCResponse, 1)
	rpc := RPC{
		RespChan: respCh,
	}

	// Decode the command
	switch rpcType {
	case rpcSync:
		var req SyncRequest
		if err := dec.Decode(&req); err != nil {
			return err
		}
		rpc.Command = &req
	case rpcEagerSync:
		var req EagerSyncRequest
		if err := dec.Decode(&req); err != nil {
			return err
		}
		rpc.Command = &req
	case rpcFastForward:
		var req FastForwardRequest
		if err := dec.Decode(&req); err != nil {
			return err
		}
		rpc.Command = &req
	case rpcJoin:
		var req JoinRequest
		if err := dec.Decode(&req); err != nil {
			return err
		}
		rpc.Command = &req
	default:
		return fmt.Errorf("unknown rpc type %d", rpcType)
	}

	// Dispatch the RPC
	select {
	case n.consumeCh <- rpc:
	case <-n.shutdownCh:
		return ErrTransportShutdown
	}

	// Wait for response
	select {
	case resp := <-respCh:
		// Send the error first
		respErr := ""
		if resp.Error != nil {
			respErr = resp.Error.Error()
		}
		if err := enc.Encode(respErr); err != nil {
			return err
		}

		// Send the response
		if err := enc.Encode(resp.Response); err != nil {
			return err
		}
	case <-n.shutdownCh:
		return ErrTransportShutdown
	}

	return nil
}
