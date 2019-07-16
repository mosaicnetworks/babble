package hashgraph

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/crypto/keys"
	"github.com/mosaicnetworks/babble/src/peers"
)

/*******************************************************************************
InternalTransactionBody
*******************************************************************************/

// TransactionType ...
type TransactionType uint8

const (
	// PEER_ADD ...
	PEER_ADD TransactionType = iota
	// PEER_REMOVE ...
	PEER_REMOVE
)

// String ...
func (t TransactionType) String() string {
	switch t {
	case PEER_ADD:
		return "PEER_ADD"
	case PEER_REMOVE:
		return "PEER_REMOVE"
	default:
		return "Unknown TransactionType"
	}
}

// InternalTransactionBody ...
type InternalTransactionBody struct {
	Type TransactionType
	Peer peers.Peer
}

//Marshal - json encoding of body
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

// InternalTransaction ...
type InternalTransaction struct {
	Body      InternalTransactionBody
	Signature string
}

// NewInternalTransaction ...
func NewInternalTransaction(tType TransactionType, peer peers.Peer) InternalTransaction {
	return InternalTransaction{
		Body: InternalTransactionBody{Type: tType, Peer: peer},
	}
}

// NewInternalTransactionJoin ...
func NewInternalTransactionJoin(peer peers.Peer) InternalTransaction {
	return NewInternalTransaction(PEER_ADD, peer)
}

// NewInternalTransactionLeave ...
func NewInternalTransactionLeave(peer peers.Peer) InternalTransaction {
	return NewInternalTransaction(PEER_REMOVE, peer)
}

// Marshal ...
func (t *InternalTransaction) Marshal() ([]byte, error) {
	var b bytes.Buffer

	enc := json.NewEncoder(&b) //will write to b

	if err := enc.Encode(t); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// Unmarshal ...
func (t *InternalTransaction) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)

	dec := json.NewDecoder(b) //will read from b

	if err := dec.Decode(t); err != nil {
		return err
	}

	return nil
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

// Verify ...
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

//AsAccepted returns a receipt to accept an InternalTransaction
func (t *InternalTransaction) AsAccepted() InternalTransactionReceipt {
	return InternalTransactionReceipt{
		InternalTransaction: *t,
		Accepted:            true,
	}
}

//AsRefused return a receipt to refuse an InternalTransaction
func (t *InternalTransaction) AsRefused() InternalTransactionReceipt {
	return InternalTransactionReceipt{
		InternalTransaction: *t,
		Accepted:            false,
	}
}

/*******************************************************************************
InternalTransactionReceipt
*******************************************************************************/

//InternalTransactionReceipt records the decision by the application to accept
//or refuse and InternalTransaction
type InternalTransactionReceipt struct {
	InternalTransaction InternalTransaction
	Accepted            bool
}
