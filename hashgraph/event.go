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

	"bitbucket.org/mosaicnet/babble/crypto"
)

type EventBody struct {
	Transactions [][]byte  //the payload
	Parents      []string  //hashes of the event's parents, self-parent first
	Creator      []byte    //creator's public key
	Timestamp    time.Time //creator's claimed timestamp of the event's creation
	Index        int       //index in the sequence of events created by Creator
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

type EventCoordinates struct {
	hash  string
	index int
}

type Event struct {
	Body EventBody
	R, S *big.Int //creator's digital signature of body

	roundReceived      *int
	consensusTimestamp time.Time

	lastAncestors    []EventCoordinates
	firstDescendants []EventCoordinates
}

func NewEvent(transactions [][]byte,
	parents []string,
	creator []byte,
	index int) Event {

	body := EventBody{
		Transactions: transactions,
		Parents:      parents,
		Creator:      creator,
		Timestamp:    time.Now(),
		Index:        index,
	}
	return Event{
		Body: body,
	}
}

func (e *Event) Creator() string {
	return fmt.Sprintf("0x%X", e.Body.Creator)
}

func (e *Event) SelfParent() string {
	return e.Body.Parents[0]
}

func (e *Event) OtherParent() string {
	return e.Body.Parents[1]
}

func (e *Event) Transactions() [][]byte {
	return e.Body.Transactions
}

func (e *Event) Index() int {
	return e.Body.Index
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

func (e *Event) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := gob.NewDecoder(b) //will read from b
	return dec.Decode(e)
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

func (e *Event) SetRoundReceived(rr int) {
	if e.roundReceived == nil {
		e.roundReceived = new(int)
	}
	*e.roundReceived = rr
}

//Sorting

// ByTimestamp implements sort.Interface for []Event based on
// the timestamp field.
type ByTimestamp []Event

func (a ByTimestamp) Len() int           { return len(a) }
func (a ByTimestamp) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTimestamp) Less(i, j int) bool { return a[i].Body.Timestamp.Sub(a[j].Body.Timestamp) < 0 }
