package node

import (
	"fmt"
	"time"

	"github.com/mosaicnetworks/babble/src/hashgraph"
	hg "github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/mosaicnetworks/babble/src/net"
	_state "github.com/mosaicnetworks/babble/src/node/state"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/sirupsen/logrus"
)

func (n *Node) requestSync(target string, known map[uint32]int, syncLimit int) (net.SyncResponse, error) {
	args := net.SyncRequest{
		FromID:    n.core.validator.ID(),
		SyncLimit: syncLimit,
		Known:     known,
	}

	var out net.SyncResponse

	err := n.trans.Sync(target, &args, &out)

	return out, err
}

func (n *Node) requestEagerSync(target string, events []hg.WireEvent) (net.EagerSyncResponse, error) {
	args := net.EagerSyncRequest{
		FromID: n.core.validator.ID(),
		Events: events,
	}

	var out net.EagerSyncResponse

	err := n.trans.EagerSync(target, &args, &out)

	return out, err
}

func (n *Node) requestFastForward(target string) (net.FastForwardResponse, error) {
	n.logger.WithFields(logrus.Fields{
		"target": target,
	}).Debug("RequestFastForward()")

	args := net.FastForwardRequest{
		FromID: n.core.validator.ID(),
	}

	var out net.FastForwardResponse

	err := n.trans.FastForward(target, &args, &out)

	return out, err
}

func (n *Node) requestJoin(target string) (net.JoinResponse, error) {

	joinTx := hashgraph.NewInternalTransactionJoin(*peers.NewPeer(
		n.core.validator.PublicKeyHex(),
		n.trans.AdvertiseAddr(),
		n.core.validator.Moniker))

	joinTx.Sign(n.core.validator.Key)

	args := net.JoinRequest{InternalTransaction: joinTx}

	var out net.JoinResponse

	err := n.trans.Join(target, &args, &out)

	return out, err
}

func (n *Node) processRPC(rpc net.RPC) {

	// Notify others that we are not in Babbling state to prevent
	// them from hitting timeouts. We also allow SyncRequests while Suspended
	// because it enables the other nodes to be notified of this suspension.
	_, isSyncRequest := rpc.Command.(*net.SyncRequest)

	if state := n.GetState(); !(state == _state.Babbling ||
		(state == _state.Suspended && isSyncRequest)) {

		n.logger.WithField("state", state).Debug("Not in Babbling state")
		rpc.Respond(nil, fmt.Errorf("Not in Babbling state"))
		return
	}

	switch cmd := rpc.Command.(type) {
	case *net.SyncRequest:
		n.processSyncRequest(rpc, cmd)
	case *net.EagerSyncRequest:
		n.processEagerSyncRequest(rpc, cmd)
	case *net.FastForwardRequest:
		n.processFastForwardRequest(rpc, cmd)
	case *net.JoinRequest:
		n.processJoinRequest(rpc, cmd)
	default:
		n.logger.WithField("cmd", rpc.Command).Error("Unexpected RPC command")
		rpc.Respond(nil, fmt.Errorf("unexpected command"))
	}
}

func (n *Node) processSyncRequest(rpc net.RPC, cmd *net.SyncRequest) {
	n.logger.WithFields(logrus.Fields{
		"from_id":    cmd.FromID,
		"sync_limit": cmd.SyncLimit,
		"known":      cmd.Known,
	}).Debug("process SyncRequest")

	resp := &net.SyncResponse{
		FromID: n.core.validator.ID(),
	}

	var respErr error

	//Compute Diff
	start := time.Now()
	n.coreLock.Lock()
	eventDiff, err := n.core.EventDiff(cmd.Known)
	n.coreLock.Unlock()
	elapsed := time.Since(start)

	n.logger.WithField("duration", elapsed.Nanoseconds()).Debug("Diff()")

	if err != nil {
		n.logger.WithField("error", err).Error("Calculating Diff")
		respErr = err
	}

	if len(eventDiff) > 0 {

		//select min(cmd.SyncLimit, this.SyncLimit) events
		limit := min(cmd.SyncLimit, n.conf.SyncLimit)

		n.logger.WithFields(logrus.Fields{
			"req.sync_limit": cmd.SyncLimit,
			"own.sync_limit": n.conf.SyncLimit,
			"diff_length":    len(eventDiff),
		}).Debugf("Selecting max %d events", limit)

		if limit < len(eventDiff) {
			eventDiff = eventDiff[:limit]
		}

		//Convert to WireEvents
		wireEvents, err := n.core.ToWire(eventDiff)
		if err != nil {
			n.logger.WithField("error", err).Debug("Converting to WireEvent")
			respErr = err
		} else {
			resp.Events = wireEvents
		}
	}

	//Get Self Known
	n.coreLock.Lock()
	knownEvents := n.core.KnownEvents()
	n.coreLock.Unlock()

	resp.Known = knownEvents

	n.logger.WithFields(logrus.Fields{
		"events":  len(resp.Events),
		"known":   resp.Known,
		"rpc_err": respErr,
	}).Debug("Responding to SyncRequest")

	rpc.Respond(resp, respErr)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (n *Node) processEagerSyncRequest(rpc net.RPC, cmd *net.EagerSyncRequest) {
	n.logger.WithFields(logrus.Fields{
		"from_id": cmd.FromID,
		"events":  len(cmd.Events),
	}).Debug("EagerSyncRequest")

	success := true

	n.coreLock.Lock()
	err := n.sync(cmd.FromID, cmd.Events)
	n.coreLock.Unlock()

	if err != nil {
		n.logger.WithField("error", err).Error("sync()")
		success = false
	}

	resp := &net.EagerSyncResponse{
		FromID:  n.core.validator.ID(),
		Success: success,
	}

	rpc.Respond(resp, err)
}

func (n *Node) processFastForwardRequest(rpc net.RPC, cmd *net.FastForwardRequest) {
	n.logger.WithFields(logrus.Fields{
		"from": cmd.FromID,
	}).Debug("process FastForwardRequest")

	resp := &net.FastForwardResponse{
		FromID: n.core.validator.ID(),
	}

	var respErr error

	//Get latest Frame
	n.coreLock.Lock()
	block, frame, err := n.core.GetAnchorBlockWithFrame()
	n.coreLock.Unlock()

	if err != nil {
		n.logger.WithError(err).Error("Getting Frame")
		respErr = err
	} else {
		resp.Block = *block
		resp.Frame = *frame

		//Get snapshot
		snapshot, err := n.proxy.GetSnapshot(block.Index())

		if err != nil {
			n.logger.WithField("error", err).Error("Getting Snapshot")
			respErr = err
		} else {
			resp.Snapshot = snapshot
		}
	}

	n.logger.WithFields(logrus.Fields{
		"events":         len(resp.Frame.Events),
		"block":          resp.Block.Index(),
		"round_received": resp.Block.RoundReceived(),
		"rpc_err":        respErr,
	}).Debug("Responding to FastForwardRequest")

	rpc.Respond(resp, respErr)
}

func (n *Node) processJoinRequest(rpc net.RPC, cmd *net.JoinRequest) {
	n.logger.WithFields(logrus.Fields{
		"peer": cmd.InternalTransaction.Body.Peer,
	}).Debug("process JoinRequest")

	var respErr error
	var accepted bool
	var acceptedRound int
	var peers []*peers.Peer

	if ok, _ := cmd.InternalTransaction.Verify(); !ok {

		msg := "Unable to verify signature on join request"
		n.logger.Debug(msg)
		respErr = fmt.Errorf(msg)

	} else if _, ok := n.core.peers.ByPubKey[cmd.InternalTransaction.Body.Peer.PubKeyString()]; ok {

		n.logger.Debug("JoinRequest peer is already present")

		accepted = true

		//Get current peerset and accepted round
		lastConsensusRound := n.core.GetLastConsensusRoundIndex()
		if lastConsensusRound != nil {
			acceptedRound = *lastConsensusRound
		}

		peers = n.core.peers.Peers

	} else {
		// Dispatch the InternalTransaction
		n.coreLock.Lock()
		promise := n.core.AddInternalTransaction(cmd.InternalTransaction)
		n.coreLock.Unlock()

		//Wait for the InternalTransaction to go through consensus
		timeout := time.After(n.conf.JoinTimeout)
		select {
		case resp := <-promise.RespCh:
			accepted = resp.Accepted
			acceptedRound = resp.AcceptedRound
			peers = resp.Peers
		case <-timeout:
			respErr = fmt.Errorf("Timeout waiting for JoinRequest to go through consensus")
			n.logger.WithError(respErr).Error()
			break
		}
	}

	resp := &net.JoinResponse{
		FromID:        n.core.validator.ID(),
		Accepted:      accepted,
		AcceptedRound: acceptedRound,
		Peers:         peers,
	}

	n.logger.WithFields(logrus.Fields{
		"accepted":       resp.Accepted,
		"accepted_round": resp.AcceptedRound,
		"peers":          len(resp.Peers),
		"rpc_err":        respErr,
	}).Debug("Responding to JoinRequest")

	rpc.Respond(resp, respErr)
}
