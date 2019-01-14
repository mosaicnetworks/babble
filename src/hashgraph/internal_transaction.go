package hashgraph

import (
	"bytes"
	"encoding/json"
	"fmt"

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
	Accepted Trilean
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

func (t *InternalTransaction) Hash() string {
	hashBytes, _ := t.Marshal()
	hash := crypto.SHA256(hashBytes)
	return fmt.Sprintf("0x%x", hash)
}

func (t *InternalTransaction) Accept() {
	t.Accepted = True
}

func (t *InternalTransaction) Refuse() {
	t.Accepted = False
}
