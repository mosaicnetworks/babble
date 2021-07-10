package net

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/config"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/net/signal/wamp"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/nknorg/nkn-sdk-go"
)

const (
	NKN               = iota
	numTestTransports // NOTE: must be last
	INMEM             // NOTE: move up to include in tests
	TCP               // NOTE: move up to include in tests
	WEBRTC            // NOTE: move up to include in tests
)

var (
	realm             = config.DefaultSignalRealm
	wampPort          = 8443
	certFile          = "signal/wamp/test_data/cert.pem"
	keyFile           = "signal/wamp/test_data/key.pem"
	signalTimeout     = 5 * time.Second
	nknConnectTimeout = 10 * time.Second
	listenTimeout     = 20 * time.Second
)

func NewTestTransport(ttype int, addr string, wampserver string, t *testing.T) Transport {
	switch ttype {
	case INMEM:
		_, it := NewInmemTransport(addr)
		return it
	case TCP:
		tt, err := NewTCPTransport(
			addr,
			"",
			2,
			time.Second,
			2*time.Second,
			common.NewTestEntry(t, common.TestLogLevel).WithField("node", addr),
		)
		if err != nil {
			t.Fatal(err)
		}
		go tt.Listen()
		return tt
	case WEBRTC:
		signal, err := wamp.NewClient(
			wampserver,
			realm,
			addr,
			certFile,
			false,
			signalTimeout,
			common.NewTestEntry(t, common.TestLogLevel),
		)
		if err != nil {
			t.Fatal(err)
		}
		wt, err := NewWebRTCTransport(
			signal,
			config.DefaultICEServers(),
			1,
			signalTimeout,
			signalTimeout,
			common.NewTestEntry(t, common.TestLogLevel))
		if err != nil {
			t.Fatal(err)
		}
		go wt.Listen()
		return wt
	case NKN:
		account, err := nkn.NewAccount(nil)
		if err != nil {
			t.Fatal(err)
		}
		nt, err := NewNKNTransport(
			account,
			"babble",
			10,
			nil,
			nknConnectTimeout,
			1,
			20*time.Second,
			20*time.Second,
			common.NewTestEntry(t, common.TestLogLevel))
		if err != nil {
			t.Fatal(err)
		}
		go nt.Listen()
		return nt
	default:
		panic("Unknown transport type")
	}
}

func checkStartWampServer(ttype int, address string, t *testing.T) *wamp.Server {
	if ttype == WEBRTC {
		time.Sleep(time.Second)
		server, err := wamp.NewServer(address, realm, certFile, keyFile, common.NewTestEntry(t, common.TestLogLevel))
		if err != nil {
			t.Fatal(err)
		}
		return server
	}
	return nil
}

func nextWampAddress() string {
	res := fmt.Sprintf("localhost:%d", wampPort)
	wampPort++
	return res
}

func TestTransport_StartStop(t *testing.T) {
	wampserver := nextWampAddress()

	for ttype := 0; ttype < numTestTransports; ttype++ {
		s := checkStartWampServer(ttype, wampserver, t)
		if s != nil {
			go s.Run()
			defer s.Shutdown()
			time.Sleep(time.Second)
		}

		trans := NewTestTransport(ttype, "127.0.0.1:0", wampserver, t)
		if err := trans.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}
}

func TestTransport_Sync(t *testing.T) {
	wampserver := nextWampAddress()
	addr1 := "127.0.0.1:1234"
	addr2 := "127.0.0.1:1235"

	for ttype := 0; ttype < numTestTransports; ttype++ {
		s := checkStartWampServer(ttype, wampserver, t)
		if s != nil {
			go s.Run()
			defer s.Shutdown()
			time.Sleep(time.Second)
		}

		trans1 := NewTestTransport(ttype, addr1, wampserver, t)
		defer trans1.Close()
		rpcCh := trans1.Consumer()

		// Make the RPC request
		args := SyncRequest{
			FromID:    0,
			SyncLimit: 20,
			Known: map[uint32]int{
				0: 1,
				1: 2,
				2: 3,
			},
		}
		resp := SyncResponse{
			FromID: 1,
			Events: []hashgraph.WireEvent{
				{
					Body: hashgraph.WireBody{
						Transactions:         [][]byte(nil),
						SelfParentIndex:      1,
						OtherParentCreatorID: 10,
						OtherParentIndex:     0,
						CreatorID:            9,
					},
				},
			},
			Known: map[uint32]int{
				0: 5,
				1: 5,
				2: 6,
			},
		}

		// Listen for a request
		stopCh := make(chan struct{})
		defer close(stopCh)
		go func() {
			select {
			case rpc := <-rpcCh:
				// Verify the command
				req := rpc.Command.(*SyncRequest)
				if !reflect.DeepEqual(req, &args) {
					t.Logf("command mismatch: %#v %#v", *req, args)
				}
				rpc.Respond(&resp, nil)
			case <-stopCh:
				return
			case <-time.After(listenTimeout):
				t.Logf("consumer timeout")
			}
		}()

		// Transport 2 makes outbound request
		trans2 := NewTestTransport(ttype, addr2, wampserver, t)
		defer trans2.Close()

		if ttype == INMEM {
			itrans1 := trans1.(*InmemTransport)
			itrans2 := trans2.(*InmemTransport)
			itrans1.Connect(addr2, trans2)
			itrans2.Connect(addr1, trans1)
			trans1 = itrans1
			trans2 = itrans2
		}

		var out SyncResponse
		if err := trans2.Sync(trans1.AdvertiseAddr(), &args, &out); err != nil {
			t.Fatalf("err: %v", err)
		}

		// Verify the response
		if !reflect.DeepEqual(resp, out) {
			t.Fatalf("command mismatch: %#v %#v", resp, out)
		}
	}
}

func TestTransport_EagerSync(t *testing.T) {
	wampserver := nextWampAddress()
	addr1 := "127.0.0.1:1236"
	addr2 := "127.0.0.1:1237"

	for ttype := 0; ttype < numTestTransports; ttype++ {
		s := checkStartWampServer(ttype, wampserver, t)
		if s != nil {
			go s.Run()
			defer s.Shutdown()
			time.Sleep(time.Second)
		}

		trans1 := NewTestTransport(ttype, addr1, wampserver, t)
		defer trans1.Close()
		rpcCh := trans1.Consumer()

		// Make the RPC request
		args := EagerSyncRequest{
			FromID: 0,
			Events: []hashgraph.WireEvent{
				{
					Body: hashgraph.WireBody{
						Transactions:         [][]byte(nil),
						SelfParentIndex:      1,
						OtherParentCreatorID: 10,
						OtherParentIndex:     0,
						CreatorID:            9,
					},
				},
			},
		}
		resp := EagerSyncResponse{
			FromID:  1,
			Success: true,
		}

		// Listen for a request
		stopCh := make(chan struct{})
		defer close(stopCh)
		go func() {
			select {
			case rpc := <-rpcCh:
				// Verify the command
				req := rpc.Command.(*EagerSyncRequest)
				if !reflect.DeepEqual(req, &args) {
					t.Logf("command mismatch: %#v %#v", *req, args)
				}
				rpc.Respond(&resp, nil)
			case <-stopCh:
				return
			case <-time.After(listenTimeout):
				t.Logf("consumer timeout")
			}
		}()

		// Transport 2 makes outbound request
		trans2 := NewTestTransport(ttype, addr2, wampserver, t)
		defer trans2.Close()

		if ttype == INMEM {
			itrans1 := trans1.(*InmemTransport)
			itrans2 := trans2.(*InmemTransport)
			itrans1.Connect(addr2, trans2)
			itrans2.Connect(addr1, trans1)
			trans1 = itrans1
			trans2 = itrans2
		}

		var out EagerSyncResponse
		if err := trans2.EagerSync(trans1.AdvertiseAddr(), &args, &out); err != nil {
			t.Fatalf("err: %v", err)
		}

		// Verify the response
		if !reflect.DeepEqual(resp, out) {
			t.Fatalf("command mismatch: %#v %#v", resp, out)
		}
	}
}

func TestTransport_FastForward(t *testing.T) {
	wampserver := nextWampAddress()
	addr1 := "127.0.0.1:1238"
	addr2 := "127.0.0.1:1239"

	for ttype := 0; ttype < numTestTransports; ttype++ {
		s := checkStartWampServer(ttype, wampserver, t)
		if s != nil {
			go s.Run()
			defer s.Shutdown()
			time.Sleep(time.Second)
		}

		trans1 := NewTestTransport(ttype, addr1, wampserver, t)
		defer trans1.Close()
		rpcCh := trans1.Consumer()

		//Prepare the response Frame and corresponding Block

		framePeers := []*peers.Peer{
			peers.NewPeer("pub1", "addr1", "monika"),
			peers.NewPeer("pub2", "addr2", "monika"),
		}

		//Marshalling/Unmarshalling clears private fields, so we precompute the
		//Marsalled/Unmarshalled objects to compare the expected result to the
		//RPC response.

		frame := &hashgraph.Frame{
			Round: 10,
			Peers: framePeers,
			Roots: map[string]*hashgraph.Root{
				"pub1": hashgraph.NewRoot(),
				"pub2": hashgraph.NewRoot(),
			},
			Events: []*hashgraph.FrameEvent{
				{
					Core: hashgraph.NewEvent(
						[][]byte{
							[]byte("tx1"),
							[]byte("tx2"),
						},
						[]hashgraph.InternalTransaction{
							hashgraph.NewInternalTransaction(hashgraph.PEER_ADD, *peers.NewPeer("pub3", "addr3", "monika")),
						},
						[]hashgraph.BlockSignature{
							{
								Validator: []byte("pub1"),
								Index:     0,
								Signature: "the signature",
							},
						},
						[]string{"pub1", "pub2"},
						[]byte("pub1"),
						4,
					),
				},
			},
		}

		marshalledFrame, err := frame.Marshal()
		if err != nil {
			t.Fatal(err)
		}

		var unmarshalledFrame hashgraph.Frame
		err = unmarshalledFrame.Unmarshal(marshalledFrame)
		if err != nil {
			t.Fatal(err)
		}

		block, err := hashgraph.NewBlockFromFrame(9, frame)
		if err != nil {
			t.Fatal(err)
		}

		marshalledBlock, err := block.Marshal()
		if err != nil {
			t.Fatal(err)
		}

		var unmarshalledBlock hashgraph.Block
		err = unmarshalledBlock.Unmarshal(marshalledBlock)
		if err != nil {
			t.Fatal(err)
		}

		snapshot := []byte("this is the snapshot")

		// Make the RPC request and response

		args := FastForwardRequest{
			FromID: 0,
		}
		resp := FastForwardResponse{
			FromID:   1,
			Block:    unmarshalledBlock,
			Frame:    unmarshalledFrame,
			Snapshot: snapshot,
		}

		// Listen for a request
		stopCh := make(chan struct{})
		defer close(stopCh)
		go func() {
			select {
			case rpc := <-rpcCh:
				// Verify the command
				req := rpc.Command.(*FastForwardRequest)
				if !reflect.DeepEqual(req, &args) {
					t.Logf("command mismatch: %#v %#v", *req, args)
				}
				rpc.Respond(&resp, nil)
			case <-stopCh:
				return
			case <-time.After(listenTimeout):
				t.Logf("consumer timeout")
			}
		}()

		// Transport 2 makes outbound request
		trans2 := NewTestTransport(ttype, addr2, wampserver, t)
		defer trans2.Close()

		if ttype == INMEM {
			itrans1 := trans1.(*InmemTransport)
			itrans2 := trans2.(*InmemTransport)
			itrans1.Connect(addr2, trans2)
			itrans2.Connect(addr1, trans1)
			trans1 = itrans1
			trans2 = itrans2
		}

		var out FastForwardResponse
		if err := trans2.FastForward(trans1.AdvertiseAddr(), &args, &out); err != nil {
			t.Fatalf("err: %v", err)
		}

		// Verify the response
		if !reflect.DeepEqual(resp, out) {
			t.Fatalf("ttype %d. Response mismatch: %#v %#v", ttype, resp, out)
		}
	}
}

func TestTransport_Join(t *testing.T) {
	wampserver := nextWampAddress()
	addr1 := "127.0.0.1:1240"
	addr2 := "127.0.0.1:1241"

	for ttype := 0; ttype < numTestTransports; ttype++ {
		s := checkStartWampServer(ttype, wampserver, t)
		if s != nil {
			go s.Run()
			defer s.Shutdown()
			time.Sleep(time.Second)
		}

		trans1 := NewTestTransport(ttype, addr1, wampserver, t)
		defer trans1.Close()
		rpcCh := trans1.Consumer()

		//node1 asks to join node2
		testPeers := []*peers.Peer{
			peers.NewPeer("node1", "addr1", "monika"),
			peers.NewPeer("node2", "addr2", "monika"),
		}

		unmarshalledPeers := []*peers.Peer{}
		for _, p := range testPeers {
			mp, err := p.Marshal()
			if err != nil {
				t.Fatal(err)
			}

			var up peers.Peer
			err = up.Unmarshal(mp)
			if err != nil {
				t.Fatal(err)
			}

			unmarshalledPeers = append(unmarshalledPeers, &up)
		}

		// Make the RPC request
		itx := hashgraph.NewInternalTransactionJoin(*unmarshalledPeers[0])
		//itx.Sign()
		args := JoinRequest{
			InternalTransaction: itx,
		}

		resp := JoinResponse{
			FromID:        testPeers[1].ID(),
			AcceptedRound: 5,
			Peers:         unmarshalledPeers,
		}

		// Listen for a request
		stopCh := make(chan struct{})
		defer close(stopCh)
		go func() {
			select {
			case rpc := <-rpcCh:
				// Verify the command
				req := rpc.Command.(*JoinRequest)
				if !reflect.DeepEqual(req, &args) {
					t.Logf("command mismatch: %#v %#v", *req, args)
				}
				rpc.Respond(&resp, nil)
			case <-stopCh:
				return
			case <-time.After(listenTimeout):
				t.Logf("consumer timeout")
			}
		}()

		// Transport 2 makes outbound request
		trans2 := NewTestTransport(ttype, addr2, wampserver, t)
		defer trans2.Close()

		if ttype == INMEM {
			itrans1 := trans1.(*InmemTransport)
			itrans2 := trans2.(*InmemTransport)
			itrans1.Connect(addr2, trans2)
			itrans2.Connect(addr1, trans1)
			trans1 = itrans1
			trans2 = itrans2
		}

		var out JoinResponse
		if err := trans2.Join(trans1.AdvertiseAddr(), &args, &out); err != nil {
			t.Fatalf("err: %v", err)
		}

		// Verify the response
		if !reflect.DeepEqual(resp, out) {
			t.Fatalf("response mismatch: %#v %#v", resp, out)
		}
	}
}
