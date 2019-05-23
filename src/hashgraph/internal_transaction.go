package hashgraph

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/crypto/keys"
	"github.com/mosaicnetworks/babble/src/peers"
)

/*******************************************************************************
InternalTransactionBody
*******************************************************************************/

type TransactionType uint8

const (
	PEER_ADD TransactionType = iota
	PEER_REMOVE
)

type InternalTransactionBody struct {
	Type TransactionType
	Peer peers.Peer
}

//json encoding of body
func (i *InternalTransactionBody) Marshal() ([]byte, error) {
	var b bytes.Buffer

	enc := json.NewEncoder(&b) //will write to b

	if err := enc.Encode(i); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

//Hash returns the SHA256 hash of the InternalTransactionBody,
func (i *InternalTransactionBody) Hash() ([]byte, error) {
	hashBytes, err := i.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}

/*******************************************************************************
InternalTransaction
*******************************************************************************/

type InternalTransaction struct {
	Body      InternalTransactionBody
	Signature string
	Accepted  common.Trilean
}

func NewInternalTransaction(tType TransactionType, peer peers.Peer) InternalTransaction {
	return InternalTransaction{
		Body: InternalTransactionBody{Type: tType, Peer: peer},
	}
}

func NewInternalTransactionJoin(peer peers.Peer) InternalTransaction {
	return NewInternalTransaction(PEER_ADD, peer)
}

func NewInternalTransactionLeave(peer peers.Peer) InternalTransaction {
	return NewInternalTransaction(PEER_REMOVE, peer)
}

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

func (t *InternalTransaction) AsAccepted() InternalTransaction {
	return InternalTransaction{
		Body: InternalTransactionBody{
			Type: t.Body.Type,
			Peer: t.Body.Peer,
		},
		Signature: t.Signature,
		Accepted:  common.True,
	}
}

func (t *InternalTransaction) AsRefuse() InternalTransaction {
	return InternalTransaction{
		Body: InternalTransactionBody{
			Type: t.Body.Type,
			Peer: t.Body.Peer,
		},
		Signature: t.Signature,
		Accepted:  common.False,
	}
}

//Sign returns the ecdsa signature of the SHA256 hash of the transaction's body
func (t *InternalTransaction) Sign(privKey *ecdsa.PrivateKey) error {
	signBytes, err := t.Body.Hash()
	if err != nil {
		return err
	}

	R, S, err := keys.Sign(privKey, signBytes)
	if err != nil {
		return err
	}

	t.Signature = keys.EncodeSignature(R, S)

	return err
}

func (t *InternalTransaction) Verify() (bool, error) {
	pubBytes := t.Body.Peer.PubKeyBytes()
	pubKey := keys.ToPublicKey(pubBytes)

	signBytes, err := t.Body.Hash()
	if err != nil {
		return false, err
	}

	r, s, err := keys.DecodeSignature(t.Signature)
	if err != nil {
		return false, err
	}

	return keys.Verify(pubKey, signBytes, r, s), nil
}

//HashString returns a string representation of the body's hash. It is used in
//node/core as a key in a map to keep track of InternalTransactions as they are
//being processed asynchronously by the consensus and application.
func (t *InternalTransaction) HashString() string {
	hash, _ := t.Body.Hash()
	return string(hash)
}
