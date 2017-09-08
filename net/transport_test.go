package net

import (
	"reflect"
	"testing"
	"time"

	"bitbucket.org/mosaicnet/babble/hashgraph"
)

const (
	TTInmem = iota

	// NOTE: must be last
	numTestTransports
)

func NewTestTransport(ttype int, addr string) (string, LoopbackTransport) {
	switch ttype {
	case TTInmem:
		addr, lt := NewInmemTransport(addr)
		return addr, lt
	default:
		panic("Unknown transport type")
	}
}

func TestTransport_StartStop(t *testing.T) {
	for ttype := 0; ttype < numTestTransports; ttype++ {
		_, trans := NewTestTransport(ttype, "")
		if err := trans.Close(); err != nil {
			t.Fatalf("err: %v", err)
		}
	}
}         
func TestTransport_Sync(t *testing.T) {
	for ttype := 0; ttype < numTestTransports; ttype++ {
		addr1, trans1 := NewTestTransport(ttype, "")
		defer trans1.Close()
		rpcCh := trans1.Consumer()

		// Make the RPC request
		args := SyncRequest{
			From: "A",
			Known: map[int]int{
				0: 1,
				1: 2,
				2: 3,
			},
		}
		resp := SyncResponse{
			From: "B",
			Events: []hashgraph.WireEvent{
				hashgraph.WireEvent{
					Body: hashgraph.WireBody{
						Transactions:         [][]byte(nil),
						SelfParentIndex:      1,
						OtherParentCreatorID: 10,
						OtherParentIndex:     0,
						CreatorID:            9,
					},
				},
			},
			Known: map[int]int{
				0: 4,
				1: 5,
				2: 6,
			},
		}

		// Listen for a request
		go func() {
			select {
			case rpc := <-rpcCh:
				// Verify the command
				req := rpc.Command.(*SyncRequest)
				if !reflect.DeepEqual(req, &args) {
					t.Fatalf("command mismatch: %#v %#v", *req, args)
				}
				rpc.Respond(&resp, nil)

			case <-time.After(200 * time.Millisecond):
				t.Fatalf("timeout")
			}
		}()

		// Transport 2 makes outbound request
		addr2, trans2 := NewTestTransport(ttype, "")
		defer trans2.Close()

		trans1.Connect(addr2, trans2)
		trans2.Connect(addr1, trans1)

		var out SyncResponse
		if err := trans2.Sync(trans1.LocalAddr(), &args, &out); err != nil {
			t.Fatalf("err: %v", err)
		}

		// Verify the response
		if !reflect.DeepEqual(resp, out) {
			t.Fatalf("command mismatch: %#v %#v", resp, out)
		}
	}
}
func TestTransport_EagerSync(t *testing.T) {
	for ttype := 0; ttype < numTestTransports; ttype++ {
		addr1, trans1 := NewTestTransport(ttype, "")
		defer trans1.Close()
		rpcCh := trans1.Consumer()

		// Make the RPC request
		args := EagerSyncRequest{
			From: "A",
			Events: []hashgraph.WireEvent{
				hashgraph.WireEvent{
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
			Success: true,
		}

		// Listen for a request
		go func() {
			select {
			case rpc := <-rpcCh:
				// Verify the command
				req := rpc.Command.(*EagerSyncRequest)
				if !reflect.DeepEqual(req, &args) {
					t.Fatalf("command mismatch: %#v %#v", *req, args)
				}
				rpc.Respond(&resp, nil)

			case <-time.After(200 * time.Millisecond):
				t.Fatalf("timeout")
			}
		}()

		// Transport 2 makes outbound request
		addr2, trans2 := NewTestTransport(ttype, "")
		defer trans2.Close()

		trans1.Connect(addr2, trans2)
		trans2.Connect(addr1, trans1)

		var out EagerSyncResponse
		if err := trans2.EagerSync(trans1.LocalAddr(), &args, &out); err != nil {
			t.Fatalf("err: %v", err)
		}

		// Verify the response
		if !reflect.DeepEqual(resp, out) {
			t.Fatalf("command mismatch: %#v %#v", resp, out)
		}
	}
}
func TestTransport_FastForward(t *testing.T) {
	for ttype := 0; ttype < numTestTransports; ttype++ {
		addr1, trans1 := NewTestTransport(ttype, "")
		defer trans1.Close()
		rpcCh := trans1.Consumer()

		// Make the RPC request
		args := FastForwardRequest{
			From: "A",
		}
		resp := FastForwardResponse{
			From: "B",
			Frame: hashgraph.Frame{
				Roots: map[string]hashgraph.Root{
					"0": hashgraph.Root{
						X:     "x0",
						Y:     "y0",
						Index: 4,
						Round: 2,
						Others: map[string]string{
							"o1": "oldEvent",
						},
					},
					"1": hashgraph.Root{
						X:     "x1",
						Y:     "y1",
						Index: 4,
						Round: 2,
					},
					"2": hashgraph.Root{
						X:     "x2",
						Y:     "y2",
						Index: 4,
						Round: 2,
					},
				},
				Events: []hashgraph.Event{
					hashgraph.Event{
						Body: hashgraph.EventBody{
							Transactions: [][]byte(nil),
							Parents:      []string{"p1", "p2"},
							Creator:      []byte("creator"),
							Index:        19,
							Timestamp:    time.Now(),
						},
					},
				},
			},
		}

		// Listen for a request
		go func() {
			select {
			case rpc := <-rpcCh:
				// Verify the command
				req := rpc.Command.(*FastForwardRequest)
				if !reflect.DeepEqual(req, &args) {
					t.Fatalf("command mismatch: %#v %#v", *req, args)
				}

				rpc.Respond(&resp, nil)

			case <-time.After(200 * time.Millisecond):
				t.Fatalf("timeout")
			}
		}()

		// Transport 2 makes outbound request
		addr2, trans2 := NewTestTransport(ttype, "")
		defer trans2.Close()

		trans1.Connect(addr2, trans2)
		trans2.Connect(addr1, trans1)

		var out FastForwardResponse
		if err := trans2.FastForward(trans1.LocalAddr(), &args, &out); err != nil {
			t.Fatalf("err: %v", err)
		}

		// Verify the response
		if !reflect.DeepEqual(resp, out) {
			t.Fatalf("command mismatch: %#v %#v", resp, out)
		}
	}
}
