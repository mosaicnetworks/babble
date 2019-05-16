package hashgraph

import (
	"bytes"
	"encoding/json"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/peers"
)

/*******************************************************************************
InternalTransactions
*******************************************************************************/

type TransactionType uint8

const (
	PEER_ADD TransactionType = iota
	PEER_REMOVE
)

type InternalTransaction struct {
	Type     TransactionType
	Peer     peers.Peer
	Accepted common.Trilean
}

func NewInternalTransaction(tType TransactionType, peer peers.Peer) InternalTransaction {
	return InternalTransaction{
		Type: tType,
		Peer: peer,
	}
}

func NewInternalTransactionJoin(peer peers.Peer) InternalTransaction {
	return NewInternalTransaction(PEER_ADD, peer)
}

func NewInternalTransactionLeave(peer peers.Peer) InternalTransaction {
	return NewInternalTransaction(PEER_REMOVE, peer)
}

//json encoding of body only
func (t *InternalTransaction) Marshal() ([]byte, error) {
	var b bytes.Buffer

	enc := json.NewEncoder(&b) //will write to b

	if err := enc.Encode(t); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (t *InternalTransaction) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)

	dec := json.NewDecoder(b) //will read from b

	if err := dec.Decode(t); err != nil {
		return err
	}

	return nil
}

//Hash returns a hash of the InternalTransaction, excluding the Accepted field.
//This hash is used by /node/core as a key in a map to track internal
//transactions, so it should not include the Accepted field because it changes.
func (t *InternalTransaction) Hash() string {
	tx := InternalTransaction{
		Type: t.Type,
		Peer: t.Peer,
	}
	hashBytes, _ := tx.Marshal()
	hash := crypto.SHA256(hashBytes)
	return common.EncodeToString(hash)
}

func (t *InternalTransaction) AsAccepted() InternalTransaction {
	return InternalTransaction{
		Type:     t.Type,
		Peer:     t.Peer,
		Accepted: common.True,
	}
}

func (t *InternalTransaction) AsRefuse() InternalTransaction {
	return InternalTransaction{
		Type:     t.Type,
		Peer:     t.Peer,
		Accepted: common.False,
	}
}
