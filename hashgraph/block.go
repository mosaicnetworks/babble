package hashgraph

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/babbleio/babble/crypto"
)

type Block struct {
	RoundReceived int
	Transactions  [][]byte

	hash []byte
	hex  string
}

func NewBlock(roundReceived int, transactions [][]byte) Block {
	return Block{
		RoundReceived: roundReceived,
		Transactions:  transactions,
	}
}

func (b *Block) Marshal() ([]byte, error) {
	bf := bytes.NewBuffer([]byte{})
	enc := json.NewEncoder(bf)
	if err := enc.Encode(b); err != nil {
		return nil, err
	}
	return bf.Bytes(), nil
}

func (b *Block) Unmarshal(data []byte) error {
	bf := bytes.NewBuffer(data)
	dec := json.NewDecoder(bf)
	if err := dec.Decode(b); err != nil {
		return err
	}
	return nil
}

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

func (b *Block) Hex() string {
	if b.hex == "" {
		hash, _ := b.Hash()
		b.hex = fmt.Sprintf("0x%X", hash)
	}
	return b.hex
}
