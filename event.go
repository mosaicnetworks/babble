/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package hashgraph

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/gob"
	"fmt"
	"math/big"
	"time"

	"github.com/arrivets/go-swirlds/crypto"
)

type EventBody struct {
	Transactions [][]byte  //the payload
	Parents      []string  //hashes of the event's parents, self-parent first
	Creator      []byte    //creator's public key
	Timestamp    time.Time //creator's claimed timestamp of the event's creation
}

//gob encoding of body only
func (e *EventBody) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b) //will write to b
	if err := enc.Encode(e); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (e *EventBody) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := gob.NewDecoder(b) //will read from b
	return dec.Decode(e)
}

func (e *EventBody) Hash() ([]byte, error) {
	hashBytes, err := e.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}

type Event struct {
	Body EventBody
	R, S *big.Int //creator's digital signature of body

	round              int
	roundReceived      int
	consensusTimestamp time.Time
}

func NewEvent(transactions [][]byte, parents []string, creator []byte) Event {
	body := EventBody{
		Transactions: transactions,
		Parents:      parents,
		Creator:      creator,
		Timestamp:    time.Now(),
	}
	return Event{Body: body}
}

//ecdsa sig
func (e *Event) Sign(privKey *ecdsa.PrivateKey) error {
	signBytes, err := e.Body.Hash()
	if err != nil {
		return err
	}
	e.R, e.S, err = crypto.Sign(privKey, signBytes)
	return err
}

func (e *Event) Verify() (bool, error) {
	pubBytes := e.Body.Creator
	pubKey := crypto.ToECDSAPub(pubBytes)

	signBytes, err := e.Body.Hash()
	if err != nil {
		return false, err
	}

	return crypto.Verify(pubKey, signBytes, e.R, e.S), nil
}

//gob encoding of body and signature
func (e *Event) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := gob.NewEncoder(&b)
	if err := enc.Encode(e); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

//sha256 hash of body and signature
func (e *Event) Hash() ([]byte, error) {
	hashBytes, err := e.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}

func (e *Event) Hex() string {
	hash, _ := e.Hash()
	return fmt.Sprintf("0x%X", hash)
}

//Sorting

// ByTimestamp implements sort.Interface for []Event based on
// the timestamp field.
type ByTimestamp []Event

func (a ByTimestamp) Len() int           { return len(a) }
func (a ByTimestamp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTimestamp) Less(i, j int) bool { return a[i].Body.Timestamp.Sub(a[j].Body.Timestamp) < 0 }
