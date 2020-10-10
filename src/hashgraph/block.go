package hashgraph

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/crypto/keys"
	"github.com/mosaicnetworks/babble/src/peers"
)

// BlockBody is the content of a Block.
type BlockBody struct {
	Index                       int                          // block index
	RoundReceived               int                          // round received of corresponding hashgraph frame
	Timestamp                   int64                        // unix timestamp ((median of timestamps in round-received famous witnesses))
	StateHash                   []byte                       // root hash of the application after applying block payload; to be populated by application Commit
	FrameHash                   []byte                       // hash of corresponding hashgraph frame
	PeersHash                   []byte                       // hash of peer-set
	Transactions                [][]byte                     // transaction payload
	InternalTransactions        []InternalTransaction        // internal transaction payload (add/remove peers)
	InternalTransactionReceipts []InternalTransactionReceipt // receipts for internal transactions; to be populated by application Commit
}

// Marshal produces the JSON encoding of a BlockBody.
func (bb *BlockBody) Marshal() ([]byte, error) {
	bf := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(bf)
	if err := enc.Encode(bb); err != nil {
		return nil, err
	}
	return bf.Bytes(), nil
}

// Unmarshal parses a JSON encoded BlockBody.
func (bb *BlockBody) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) //will read from b
	if err := dec.Decode(bb); err != nil {
		return err
	}
	return nil
}

// Hash produces the SHA256 hash of the marshalled BlockBody.
func (bb *BlockBody) Hash() ([]byte, error) {
	hashBytes, err := bb.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}

// BlockSignature gathers a signature encoded as a string, along with the index
// of the block, and the public-key of the signer.
type BlockSignature struct {
	// Validator is the public key of the signer.
	Validator []byte
	// Block Index
	Index int
	// String encoding of the signature
	Signature string
}

// ValidatorHex returns the hex string representation of the signer's public
// key.
func (bs *BlockSignature) ValidatorHex() string {
	return common.EncodeToString(bs.Validator)
}

// Marshal produces the JSON encoding of the BlockSignature.
func (bs *BlockSignature) Marshal() ([]byte, error) {
	bf := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(bf)
	if err := enc.Encode(bs); err != nil {
		return nil, err
	}
	return bf.Bytes(), nil
}

// Unmarshal parses a BlockSignature from JSON.
func (bs *BlockSignature) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) //will read from b
	if err := dec.Decode(bs); err != nil {
		return err
	}
	return nil
}

// ToWire returns the wire representation of a BlockSignature.
func (bs *BlockSignature) ToWire() WireBlockSignature {
	return WireBlockSignature{
		Index:     bs.Index,
		Signature: bs.Signature,
	}
}

// Key produces a string identifier of the signature for storage in a key-value
// store.
func (bs *BlockSignature) Key() string {
	return fmt.Sprintf("%d-%s", bs.Index, bs.ValidatorHex())
}

// WireBlockSignature is a light-weight representation of a signature to travel
// over the wire.
type WireBlockSignature struct {
	Index     int
	Signature string
}

// Block represents a section of the Hashgraph that has reached consensus. It
// contains an ordered list of transactions and internal transactions. When a
// section of the hashgraph reaches consensus, a corresponding block is
// assembled and committed to the application. After processing the block, the
// application updates the block's state-hash and internal transaction receipts.
// The block is then signed and the signatures are gossipped and gathered as
// part of the normal gossip routines. A block with enough signatures is final;
// it is a self-contained proof that a section of the hashgraph and the
// associated transactions are a result of consensus among the prevailing peer-
// set.
type Block struct {
	Body       BlockBody
	Signatures map[string]string // [validator hex] => signature

	hash    []byte
	hex     string
	peerSet *peers.PeerSet
}

// NewBlockFromFrame assembles a block from a Frame.
func NewBlockFromFrame(blockIndex int, frame *Frame) (*Block, error) {
	frameHash, err := frame.Hash()
	if err != nil {
		return nil, err
	}

	transactions := [][]byte{}
	internalTransactions := []InternalTransaction{}
	for _, e := range frame.Events {
		transactions = append(transactions, e.Core.Transactions()...)
		internalTransactions = append(internalTransactions, e.Core.InternalTransactions()...)
	}

	block := NewBlock(
		blockIndex,
		frame.Round,
		frameHash,
		frame.Peers,
		transactions,
		internalTransactions,
		frame.Timestamp)

	return block, nil
}

// NewBlock creates a new Block.
func NewBlock(blockIndex,
	roundReceived int,
	frameHash []byte,
	peerSlice []*peers.Peer,
	txs [][]byte,
	itxs []InternalTransaction,
	timestamp int64) *Block {

	peerSet := peers.NewPeerSet(peerSlice)

	peersHash, err := peerSet.Hash()
	if err != nil {
		return nil
	}

	body := BlockBody{
		Index:                blockIndex,
		RoundReceived:        roundReceived,
		StateHash:            []byte{},
		FrameHash:            frameHash,
		PeersHash:            peersHash,
		Transactions:         txs,
		InternalTransactions: itxs,
		Timestamp:            timestamp,
	}

	return &Block{
		Body:       body,
		Signatures: make(map[string]string),
		peerSet:    peerSet,
	}
}

// Index returns the block's index.
func (b *Block) Index() int {
	return b.Body.Index
}

// Timestamp returns the block's timestamp
func (b *Block) Timestamp() int64 {
	return b.Body.Timestamp
}

// Transactions return's the block's transactoins.
func (b *Block) Transactions() [][]byte {
	return b.Body.Transactions
}

// InternalTransactions returns the block's internal transactions.
func (b *Block) InternalTransactions() []InternalTransaction {
	return b.Body.InternalTransactions
}

// InternalTransactionReceipts returns the block's internal transaction
// receipts.
func (b *Block) InternalTransactionReceipts() []InternalTransactionReceipt {
	return b.Body.InternalTransactionReceipts
}

// RoundReceived returns the block's round-received.
func (b *Block) RoundReceived() int {
	return b.Body.RoundReceived
}

// StateHash returns the block's state hash.
func (b *Block) StateHash() []byte {
	return b.Body.StateHash
}

// FrameHash returns the block's frame hash.
func (b *Block) FrameHash() []byte {
	return b.Body.FrameHash
}

// PeersHash returns the block's peers hash.
func (b *Block) PeersHash() []byte {
	return b.Body.PeersHash
}

// GetSignatures returns the block's signatures.
func (b *Block) GetSignatures() []BlockSignature {
	res := make([]BlockSignature, len(b.Signatures))
	i := 0
	for val, sig := range b.Signatures {
		validatorBytes, _ := common.DecodeFromString(val)
		res[i] = BlockSignature{
			Validator: validatorBytes,
			Index:     b.Index(),
			Signature: sig,
		}
		i++
	}
	return res
}

// GetSignature returns a block signature from a specific validator.
func (b *Block) GetSignature(validator string) (res BlockSignature, err error) {
	sig, ok := b.Signatures[validator]
	if !ok {
		return res, fmt.Errorf("signature not found")
	}

	validatorBytes, _ := common.DecodeFromString(validator)
	return BlockSignature{
		Validator: validatorBytes,
		Index:     b.Index(),
		Signature: sig,
	}, nil
}

// AppendTransactions adds a transaction to the block.
func (b *Block) AppendTransactions(txs [][]byte) {
	b.Body.Transactions = append(b.Body.Transactions, txs...)
}

// Marshal produces a JSON encoding of the Block.
func (b *Block) Marshal() ([]byte, error) {
	bf := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(bf)
	if err := enc.Encode(b); err != nil {
		return nil, err
	}
	return bf.Bytes(), nil
}

// Unmarshal parses a JSON encoded Block.
func (b *Block) Unmarshal(data []byte) error {
	bf := bytes.NewBuffer(data)
	dec := json.NewDecoder(bf)
	if err := dec.Decode(b); err != nil {
		return err
	}
	return nil
}

// Hash returns the SHA256 encoding of a marshalled block.
func (b *Block) Hash() ([]byte, error) {
	if len(b.hash) == 0 {
		hashBytes, err := b.Marshal()
		if err != nil {
			return nil, err
		}
		b.hash = crypto.SHA256(hashBytes)
	}
	return b.hash, nil
}

// Hex returns the hex string representations of the block's hash.
func (b *Block) Hex() string {
	if b.hex == "" {
		hash, _ := b.Hash()
		b.hex = common.EncodeToString(hash)
	}
	return b.hex
}

// Sign returns the signature of the hash of the block's body.
func (b *Block) Sign(privKey *ecdsa.PrivateKey) (bs BlockSignature, err error) {
	signBytes, err := b.Body.Hash()
	if err != nil {
		return bs, err
	}
	R, S, err := keys.Sign(privKey, signBytes)
	if err != nil {
		return bs, err
	}
	signature := BlockSignature{
		Validator: keys.FromPublicKey(&privKey.PublicKey),
		Index:     b.Index(),
		Signature: keys.EncodeSignature(R, S),
	}

	return signature, nil
}

// SetSignature appends a signature to the block.
func (b *Block) SetSignature(bs BlockSignature) error {
	b.Signatures[bs.ValidatorHex()] = bs.Signature
	return nil
}

// Verify verifies that a signature is valid against the block.
func (b *Block) Verify(sig BlockSignature) (bool, error) {
	signBytes, err := b.Body.Hash()
	if err != nil {
		return false, err
	}

	pubKey := keys.ToPublicKey(sig.Validator)

	r, s, err := keys.DecodeSignature(sig.Signature)
	if err != nil {
		return false, err
	}

	return keys.Verify(pubKey, signBytes, r, s), nil
}
