package hashgraph

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/babbleio/babble/crypto"
)

type BlockBody struct {
	RoundReceived int
	Transactions  [][]byte
}

//json encoding of body only
func (bb *BlockBody) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b) //will write to b
	if err := enc.Encode(bb); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (bb *BlockBody) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) //will read from b
	if err := dec.Decode(bb); err != nil {
		return err
	}
	return nil
}

func (bb *BlockBody) Hash() ([]byte, error) {
	hashBytes, err := bb.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}

//------------------------------------------------------------------------------

type BlockSignature struct {
	Validator     []byte
	RoundReceived int
	Signature     string
}

func (bs *BlockSignature) ValidatorHex() string {
	return fmt.Sprintf("0x%X", bs.Validator)
}

func (bs *BlockSignature) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b) //will write to b
	if err := enc.Encode(bs); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (bs *BlockSignature) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) //will read from b
	if err := dec.Decode(bs); err != nil {
		return err
	}
	return nil
}

//------------------------------------------------------------------------------

type Block struct {
	Body       BlockBody
	Signatures map[string]string // [validator hex] => signature

	hash []byte
	hex  string
}

func NewBlock(roundReceived int, transactions [][]byte) Block {
	body := BlockBody{
		RoundReceived: roundReceived,
		Transactions:  transactions,
	}
	return Block{
		Body:       body,
		Signatures: make(map[string]string),
	}
}

func (b *Block) Transactions() [][]byte {
	return b.Body.Transactions
}

func (b *Block) RoundReceived() int {
	return b.Body.RoundReceived
}

func (b *Block) GetSignature(validator string) (res BlockSignature, err error) {
	sig, ok := b.Signatures[validator]
	if !ok {
		return res, fmt.Errorf("signature not found")
	}

	validatorBytes, _ := hex.DecodeString(validator[2:])
	return BlockSignature{
		Validator:     validatorBytes,
		RoundReceived: b.RoundReceived(),
		Signature:     sig,
	}, nil
}

func (b *Block) AppendTransactions(txs [][]byte) {
	b.Body.Transactions = append(b.Body.Transactions, txs...)
}

func (b *Block) AppendSignature(sig BlockSignature) error {

	validatorHex := fmt.Sprintf("0x%X", sig.Validator)

	//do nothing if the signature is already present
	_, ok := b.Signatures[validatorHex]
	if ok {
		return nil
	}

	//check that the signature is valid
	valid, err := b.Verify(sig)
	if err != nil {
		return fmt.Errorf("Error verifying block signature: %s", err)
	}
	if !valid {
		return fmt.Errorf("Invalid block signature")
	}

	//append it to the block
	b.Signatures[validatorHex] = sig.Signature

	return nil
}

func (b *Block) Marshal() ([]byte, error) {
	var bf bytes.Buffer
	enc := json.NewEncoder(&bf)
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

func (b *Block) Sign(privKey *ecdsa.PrivateKey) (bs BlockSignature, err error) {

	signBytes, err := b.Body.Hash()
	if err != nil {
		return bs, err
	}
	R, S, err := crypto.Sign(privKey, signBytes)
	if err != nil {
		return bs, err
	}
	signature := BlockSignature{
		Validator:     crypto.FromECDSAPub(&privKey.PublicKey),
		RoundReceived: b.RoundReceived(),
		Signature:     crypto.EncodeSignature(R, S),
	}

	return signature, nil
}

func (b *Block) SetSignature(bs BlockSignature) error {
	b.Signatures[bs.ValidatorHex()] = bs.Signature
	return nil
}

func (b *Block) Verify(sig BlockSignature) (bool, error) {

	signBytes, err := b.Body.Hash()
	if err != nil {
		return false, err
	}

	pubKey := crypto.ToECDSAPub(sig.Validator)

	r, s, err := crypto.DecodeSignature(sig.Signature)
	if err != nil {
		return false, err
	}

	return crypto.Verify(pubKey, signBytes, r, s), nil
}
