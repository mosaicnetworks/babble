package node

import (
	"reflect"
	"testing"
	"time"

	"github.com/mosaicnetworks/babble/src/common"
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/net"
	dummy "github.com/mosaicnetworks/babble/src/proxy/dummy"
)

func TestProcessSync(t *testing.T) {
	keys, p := initPeers(t, 2)
	testLogger := common.NewTestLogger(t)
	config := TestConfig(t)

	//Start two nodes

	peers := p.Peers

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	genesisPeerSet := clonePeerSet(t, p.Peers)

	node0 := NewNode(config,
		NewValidator(keys[0], peers[0].Moniker),
		p,
		genesisPeerSet,
		hg.NewInmemStore(config.CacheSize),
		peer0Trans,
		dummy.NewInmemDummyClient(testLogger))
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(config,
		NewValidator(keys[1], peers[1].Moniker),
		p,
		genesisPeerSet,
		hg.NewInmemStore(config.CacheSize),
		peer1Trans,
		dummy.NewInmemDummyClient(testLogger))
	node1.Init()

	node1.RunAsync(false)

	//Manually prepare SyncRequest and expected SyncResponse

	node0KnownEvents := node0.core.KnownEvents()
	node1KnownEvents := node1.core.KnownEvents()

	unknownEvents, err := node1.core.EventDiff(node0KnownEvents)
	if err != nil {
		t.Fatal(err)
	}

	unknownWireEvents, err := node1.core.ToWire(unknownEvents)
	if err != nil {
		t.Fatal(err)
	}

	args := net.SyncRequest{
		FromID: node0.validator.ID(),
		Known:  node0KnownEvents,
	}
	expectedResp := net.SyncResponse{
		FromID: node1.validator.ID(),
		Events: unknownWireEvents,
		Known:  node1KnownEvents,
	}

	//Make actual SyncRequest and check SyncResponse

	var out net.SyncResponse
	if err := peer0Trans.Sync(peers[1].NetAddr, &args, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the response
	if expectedResp.FromID != out.FromID {
		t.Fatalf("SyncResponse.FromID should be %d, not %d", expectedResp.FromID, out.FromID)
	}

	if l := len(out.Events); l != len(expectedResp.Events) {
		t.Fatalf("SyncResponse.Events should contain %d items, not %d",
			len(expectedResp.Events), l)
	}

	for i, e := range expectedResp.Events {
		ex := out.Events[i]
		if !reflect.DeepEqual(e.Body, ex.Body) {
			t.Fatalf("SyncResponse.Events[%d] should be %v, not %v", i, e.Body,
				ex.Body)
		}
	}

	if !reflect.DeepEqual(expectedResp.Known, out.Known) {
		t.Fatalf("SyncResponse.KnownEvents should be %#v, not %#v",
			expectedResp.Known, out.Known)
	}

	node0.Shutdown()
	node1.Shutdown()
}

func TestProcessEagerSync(t *testing.T) {
	keys, p := initPeers(t, 2)
	testLogger := common.NewTestLogger(t)
	config := TestConfig(t)

	//Start two nodes

	peers := p.Peers

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	genesisPeerSet := clonePeerSet(t, p.Peers)

	node0 := NewNode(config,
		NewValidator(keys[0], peers[0].Moniker),
		p,
		genesisPeerSet,
		hg.NewInmemStore(config.CacheSize),
		peer0Trans,
		dummy.NewInmemDummyClient(testLogger))
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(config,
		NewValidator(keys[1], peers[1].Moniker),
		p,
		genesisPeerSet,
		hg.NewInmemStore(config.CacheSize),
		peer1Trans,
		dummy.NewInmemDummyClient(testLogger))
	node1.Init()

	node1.RunAsync(false)

	//Manually prepare EagerSyncRequest and expected EagerSyncResponse

	node1KnownEvents := node1.core.KnownEvents()

	unknownEvents, err := node0.core.EventDiff(node1KnownEvents)
	if err != nil {
		t.Fatal(err)
	}

	unknownWireEvents, err := node0.core.ToWire(unknownEvents)
	if err != nil {
		t.Fatal(err)
	}

	args := net.EagerSyncRequest{
		FromID: node0.validator.ID(),
		Events: unknownWireEvents,
	}
	expectedResp := net.EagerSyncResponse{
		FromID:  node1.validator.ID(),
		Success: true,
	}

	//Make actual EagerSyncRequest and check EagerSyncResponse

	var out net.EagerSyncResponse
	if err := peer0Trans.EagerSync(peers[1].NetAddr, &args, &out); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Verify the response
	if expectedResp.Success != out.Success {
		t.Fatalf("EagerSyncResponse.Sucess should be %v, not %v", expectedResp.Success, out.Success)
	}

	node0.Shutdown()
	node1.Shutdown()
}

func TestProcessFastForward(t *testing.T) {
	keys, p := initPeers(t, 2)
	testLogger := common.NewTestLogger(t)
	config := TestConfig(t)

	//Start two nodes

	peers := p.Peers

	peer0Trans, err := net.NewTCPTransport(peers[0].NetAddr, nil, 2, time.Second, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer0Trans.Close()

	genesisPeerSet := clonePeerSet(t, p.Peers)

	node0 := NewNode(config,
		NewValidator(keys[0], peers[0].Moniker),
		p,
		genesisPeerSet,
		hg.NewInmemStore(config.CacheSize),
		peer0Trans,
		dummy.NewInmemDummyClient(testLogger))
	node0.Init()

	node0.RunAsync(false)

	peer1Trans, err := net.NewTCPTransport(peers[1].NetAddr, nil, 2, time.Second, time.Second, testLogger)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer peer1Trans.Close()

	node1 := NewNode(config,
		NewValidator(keys[1], peers[1].Moniker),
		p,
		genesisPeerSet,
		hg.NewInmemStore(config.CacheSize),
		peer1Trans,
		dummy.NewInmemDummyClient(testLogger))
	node1.Init()

	node1.RunAsync(false)

	//Manually prepare FastForwardRequest. We expect a 'No Anchor Block' error

	args := net.FastForwardRequest{
		FromID: node0.validator.ID(),
	}

	//Make actual FastForwardRequest and check FastForwardResponse

	var out net.FastForwardResponse

	err = peer0Trans.FastForward(peers[1].NetAddr, &args, &out)
	if err == nil {
		t.Fatalf("FastForward request should yield 'No Anchor Block' error")
	}

	node0.Shutdown()
	node1.Shutdown()
}
