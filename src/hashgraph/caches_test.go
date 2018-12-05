package hashgraph

import (
	"fmt"
	"reflect"
	"testing"

	cm "github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/peers"
)

func TestParticipantEventsCache(t *testing.T) {
	size := 10
	testSize := 25
	participants := peers.NewPeerSet([]*peers.Peer{
		peers.NewPeer("0xaa", ""),
		peers.NewPeer("0xbb", ""),
		peers.NewPeer("0xcc", ""),
	})

	pec := NewParticipantEventsCache(size)

	for _, p := range participants.Peers {
		err := pec.AddPeer(p)
		if err != nil {
			t.Fatal(err)
		}
	}

	items := make(map[string][]string)
	for pk := range participants.ByPubKey {
		items[pk] = []string{}
	}

	for i := 0; i < testSize; i++ {
		for pk := range participants.ByPubKey {
			item := fmt.Sprintf("%s%d", pk, i)

			pec.Set(pk, item, i)

			pitems := items[pk]
			pitems = append(pitems, item)
			items[pk] = pitems
		}
	}

	// GET ITEM ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	for pk := range participants.ByPubKey {

		index1 := 9
		_, err := pec.GetItem(pk, index1)
		if err == nil || !cm.Is(err, cm.TooLate) {
			t.Fatalf("Expected ErrTooLate")
		}

		index2 := 15
		expected2 := items[pk][index2]
		actual2, err := pec.GetItem(pk, index2)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected2, actual2) {
			t.Fatalf("expected and cached not equal")
		}

		index3 := 27
		expected3 := []string{}
		actual3, err := pec.Get(pk, index3)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected3, actual3) {
			t.Fatalf("expected and cached not equal")
		}
	}

	//KNOWN ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	known := pec.Known()
	for p, k := range known {
		expectedLastIndex := testSize - 1
		if k != expectedLastIndex {
			t.Errorf("Known[%d] should be %d, not %d", p, expectedLastIndex, k)
		}
	}

	//GET ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
	for pk := range participants.ByPubKey {
		if _, err := pec.Get(pk, 0); err != nil && !cm.Is(err, cm.TooLate) {
			t.Fatalf("Skipping 0 elements should return ErrTooLate")
		}

		skipIndex := 9
		expected := items[pk][skipIndex+1:]
		cached, err := pec.Get(pk, skipIndex)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("expected and cached not equal")
		}

		skipIndex2 := 15
		expected2 := items[pk][skipIndex2+1:]
		cached2, err := pec.Get(pk, skipIndex2)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected2, cached2) {
			t.Fatalf("expected and cached not equal")
		}

		skipIndex3 := 27
		expected3 := []string{}
		cached3, err := pec.Get(pk, skipIndex3)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected3, cached3) {
			t.Fatalf("expected and cached not equal")
		}
	}
}

func TestParticipantEventsCacheEdge(t *testing.T) {
	size := 10
	testSize := 11
	participants := peers.NewPeerSet([]*peers.Peer{
		peers.NewPeer("0xaa", ""),
		peers.NewPeer("0xbb", ""),
		peers.NewPeer("0xcc", ""),
	})

	pec := NewParticipantEventsCache(size)

	for _, p := range participants.Peers {
		err := pec.AddPeer(p)
		if err != nil {
			t.Fatal(err)
		}
	}

	items := make(map[string][]string)
	for pk := range participants.ByPubKey {
		items[pk] = []string{}
	}

	for i := 0; i < testSize; i++ {
		for pk := range participants.ByPubKey {
			item := fmt.Sprintf("%s%d", pk, i)

			pec.Set(pk, item, i)

			pitems := items[pk]
			pitems = append(pitems, item)
			items[pk] = pitems
		}
	}

	for pk := range participants.ByPubKey {
		expected := items[pk][size:]
		cached, err := pec.Get(pk, size-1)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(expected, cached) {
			t.Fatalf("expected (%#v) and cached (%#v) not equal", expected, cached)
		}
	}
}

func TestPeerSetCache(t *testing.T) {
	peerSetCache := NewPeerSetCache()

	peerSet0 := peers.NewPeerSet([]*peers.Peer{
		peers.NewPeer("0xaa", ""),
		peers.NewPeer("0xbb", ""),
		peers.NewPeer("0xcc", ""),
	})

	err := peerSetCache.Set(0, peerSet0)
	if err != nil {
		t.Fatal(err)
	}

	/**************************************************************************/

	peerSet3 := peerSet0.WithNewPeer(peers.NewPeer("0xdd", ""))

	err = peerSetCache.Set(3, peerSet3)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		ps, err := peerSetCache.Get(i)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ps, peerSet0) {
			t.Fatalf("PeerSet %d should be %v, not %v", i, peerSet0, ps)
		}
	}

	for i := 3; i < 6; i++ {
		ps, err := peerSetCache.Get(i)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(ps, peerSet3) {
			t.Fatalf("PeerSet %d should be %v, not %v", i, peerSet3, ps)
		}
	}

	/**************************************************************************/

	peerSet2 := peerSet0.WithNewPeer(peers.NewPeer("0xee", ""))

	err = peerSetCache.Set(2, peerSet2)
	if err != nil {
		t.Fatal(err)
	}

	ps2, err := peerSetCache.Get(2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ps2, peerSet2) {
		t.Fatalf("PeerSet %d should be %v, not %v", 2, peerSet2, ps2)
	}

	ps3, err := peerSetCache.Get(3)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ps3, peerSet3) {
		t.Fatalf("PeerSet %d should be %v, not %v", 3, peerSet3, ps3)
	}

	/**************************************************************************/

	err = peerSetCache.Set(2, peerSet2.WithNewPeer(peers.NewPeer("broken", "")))
	if err == nil || !cm.Is(err, cm.KeyAlreadyExists) {
		t.Fatalf("Resetting PeerSet 2 should throw a KeyAlreadyExists error")
	}
}

//XXX We are not ready for this

// func TestPeerSetCacheEdge(t *testing.T) {
// 	peerSetCache := NewPeerSetCache()

// 	_, err := peerSetCache.Get(4)
// 	if err == nil || !cm.Is(err, cm.KeyNotFound) {
// 		t.Fatalf("Attempting to Get from empty PeerSetCache should throw a TooLate error")
// 	}

// 	peerSet3 := peers.NewPeerSet([]*peers.Peer{
// 		peers.NewPeer("0xaa", ""),
// 		peers.NewPeer("0xbb", ""),
// 		peers.NewPeer("0xcc", ""),
// 	})

// 	err = peerSetCache.Set(3, peerSet3)
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	_, err = peerSetCache.Get(2)
// 	if err == nil || !cm.Is(err, cm.KeyNotFound) {
// 		t.Fatalf("Attempting to Get from a Round before first known Round should throw a TooLate error")
// 	}
// }
