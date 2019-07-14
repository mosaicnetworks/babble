package hashgraph

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"

	"github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/crypto/keys"
)

/*******************************************************************************
EventBody
*******************************************************************************/

type EventBody struct {
	Transactions         [][]byte              //the payload
	InternalTransactions []InternalTransaction //peers add and removal internal consensus
	Parents              []string              //hashes of the event's parents, self-parent first
	Creator              []byte                //creator's public key
	Index                int                   //index in the sequence of events created by Creator
	BlockSignatures      []BlockSignature      //list of Block signatures signed by the Event's Creator ONLY

	//These fields are not serialized
	creatorID            uint32
	otherParentCreatorID uint32
	selfParentIndex      int
	otherParentIndex     int
}

//json encoding of body only
func (e *EventBody) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b) //will write to b
	if err := enc.Encode(e); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func (e *EventBody) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) //will read from b
	if err := dec.Decode(e); err != nil {
		return err
	}
	return nil
}

func (e *EventBody) Hash() ([]byte, error) {
	hashBytes, err := e.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}

/*******************************************************************************
EventCoordinates
*******************************************************************************/

type EventCoordinates struct {
	hash  string
	index int
}

type CoordinatesMap map[string]EventCoordinates

func NewCoordinatesMap() CoordinatesMap {
	return make(map[string]EventCoordinates)
}

func (c CoordinatesMap) Copy() CoordinatesMap {
	res := make(map[string]EventCoordinates, len(c))
	for k, v := range c {
		res[k] = v
	}
	return res
}

/*******************************************************************************
Event
*******************************************************************************/

type Event struct {
	Body      EventBody
	Signature string //creator's digital signature of body

	topologicalIndex int

	//used for sorting
	round            *int
	lamportTimestamp *int

	roundReceived *int

	lastAncestors    CoordinatesMap //[participant pubkey] => last ancestor
	firstDescendants CoordinatesMap //[participant pubkey] => first descendant

	creator string
	hash    []byte
	hex     string
}

func NewEvent(transactions [][]byte,
	internalTransactions []InternalTransaction,
	blockSignatures []BlockSignature,
	parents []string,
	creator []byte,
	index int) *Event {

	body := EventBody{
		Transactions:         transactions,
		InternalTransactions: internalTransactions,
		BlockSignatures:      blockSignatures,
		Parents:              parents,
		Creator:              creator,
		Index:                index,
	}
	return &Event{
		Body: body,
	}
}

func (e *Event) Creator() string {
	if e.creator == "" {
		e.creator = common.EncodeToString(e.Body.Creator)
	}
	return e.creator
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

func (e *Event) InternalTransactions() []InternalTransaction {
	return e.Body.InternalTransactions
}

func (e *Event) Index() int {
	return e.Body.Index
}

func (e *Event) BlockSignatures() []BlockSignature {
	return e.Body.BlockSignatures
}

//True if Event contains a payload or is the initial Event of its creator
func (e *Event) IsLoaded() bool {
	if e.Body.Index == 0 {
		return true
	}

	hasTransactions := e.Body.Transactions != nil && len(e.Body.Transactions) > 0
	hasInternalTransactions := e.Body.InternalTransactions != nil && len(e.Body.InternalTransactions) > 0

	return hasTransactions || hasInternalTransactions
}

//ecdsa sig
func (e *Event) Sign(privKey *ecdsa.PrivateKey) error {
	signBytes, err := e.Body.Hash()
	if err != nil {
		return err
	}

	R, S, err := keys.Sign(privKey, signBytes)
	if err != nil {
		return err
	}

	e.Signature = keys.EncodeSignature(R, S)

	return err
}

func (e *Event) Verify() (bool, error) {

	//first check signatures on internal transactions
	for _, itx := range e.Body.InternalTransactions {
		ok, err := itx.Verify()

		if err != nil {
			return false, err
		} else if !ok {
			return false, fmt.Errorf("invalid signature on internal transaction")
		}
	}

	//then check event signature
	pubBytes := e.Body.Creator
	pubKey := keys.ToPublicKey(pubBytes)

	signBytes, err := e.Body.Hash()
	if err != nil {
		return false, err
	}

	r, s, err := keys.DecodeSignature(e.Signature)
	if err != nil {
		return false, err
	}

	return keys.Verify(pubKey, signBytes, r, s), nil
}

//json encoding of body and signature
func (e *Event) Marshal() ([]byte, error) {
	var b bytes.Buffer

	enc := json.NewEncoder(&b)

	if err := enc.Encode(e); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func (e *Event) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)

	dec := json.NewDecoder(b) //will read from b

	return dec.Decode(e)
}

//sha256 hash of body
func (e *Event) Hash() ([]byte, error) {
	if len(e.hash) == 0 {
		hash, err := e.Body.Hash()
		if err != nil {
			return nil, err
		}

		e.hash = hash
	}

	return e.hash, nil
}

func (e *Event) Hex() string {
	if e.hex == "" {
		hash, _ := e.Hash()
		e.hex = common.EncodeToString(hash)
	}

	return e.hex
}

func (e *Event) SetRound(r int) {
	if e.round == nil {
		e.round = new(int)
	}

	*e.round = r
}

func (e *Event) GetRound() *int {
	return e.round
}

func (e *Event) SetLamportTimestamp(t int) {
	if e.lamportTimestamp == nil {
		e.lamportTimestamp = new(int)
	}

	*e.lamportTimestamp = t
}

func (e *Event) SetRoundReceived(rr int) {
	if e.roundReceived == nil {
		e.roundReceived = new(int)
	}

	*e.roundReceived = rr
}

func (e *Event) SetWireInfo(selfParentIndex int,
	otherParentCreatorID uint32,
	otherParentIndex int,
	creatorID uint32) {
	e.Body.selfParentIndex = selfParentIndex
	e.Body.otherParentCreatorID = otherParentCreatorID
	e.Body.otherParentIndex = otherParentIndex
	e.Body.creatorID = creatorID
}

func (e *Event) WireBlockSignatures() []WireBlockSignature {
	if e.Body.BlockSignatures != nil {
		wireSignatures := make([]WireBlockSignature, len(e.Body.BlockSignatures))

		for i, bs := range e.Body.BlockSignatures {
			wireSignatures[i] = bs.ToWire()
		}

		return wireSignatures
	}

	return nil
}

func (e *Event) ToWire() WireEvent {
	return WireEvent{
		Body: WireBody{
			Transactions:         e.Body.Transactions,
			InternalTransactions: e.Body.InternalTransactions,
			SelfParentIndex:      e.Body.selfParentIndex,
			OtherParentCreatorID: e.Body.otherParentCreatorID,
			OtherParentIndex:     e.Body.otherParentIndex,
			CreatorID:            e.Body.creatorID,
			Index:                e.Body.Index,
			BlockSignatures:      e.WireBlockSignatures(),
		},
		Signature: e.Signature,
	}
}

/*******************************************************************************
Sorting
*******************************************************************************/

// ByTopologicalOrder implements sort.Interface for []Event based on
// the topologicalIndex field.
// THIS IS A PARTIAL ORDER
type ByTopologicalOrder []*Event

func (a ByTopologicalOrder) Len() int      { return len(a) }
func (a ByTopologicalOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByTopologicalOrder) Less(i, j int) bool {
	return a[i].topologicalIndex < a[j].topologicalIndex
}

// ByLamportTimestamp implements sort.Interface for []Event based on
// the lamportTimestamp field.
// THIS IS A TOTAL ORDER
type ByLamportTimestamp []*Event

func (a ByLamportTimestamp) Len() int      { return len(a) }
func (a ByLamportTimestamp) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByLamportTimestamp) Less(i, j int) bool {
	it, jt := -1, -1
	if a[i].lamportTimestamp != nil {
		it = *a[i].lamportTimestamp
	}
	if a[j].lamportTimestamp != nil {
		jt = *a[j].lamportTimestamp
	}
	if it != jt {
		return it < jt
	}

	wsi, _, _ := keys.DecodeSignature(a[i].Signature)
	wsj, _, _ := keys.DecodeSignature(a[j].Signature)
	return wsi.Cmp(wsj) < 0
}

/*******************************************************************************
 WireEvent
*******************************************************************************/

type WireBody struct {
	Transactions         [][]byte
	InternalTransactions []InternalTransaction
	BlockSignatures      []WireBlockSignature

	CreatorID            uint32
	OtherParentCreatorID uint32
	Index                int
	SelfParentIndex      int
	OtherParentIndex     int
}

type WireEvent struct {
	Body      WireBody
	Signature string
}

func (we *WireEvent) BlockSignatures(validator []byte) []BlockSignature {
	if we.Body.BlockSignatures != nil {
		blockSignatures := make([]BlockSignature, len(we.Body.BlockSignatures))

		for k, bs := range we.Body.BlockSignatures {
			blockSignatures[k] = BlockSignature{
				Validator: validator,
				Index:     bs.Index,
				Signature: bs.Signature,
			}
		}

		return blockSignatures
	}

	return nil
}

/*******************************************************************************
FrameEvent
******************************************************************************/

//FrameEvent is a wrapper around a regular Event. It contains exported fields
//Round, Witness, and LamportTimestamp.
type FrameEvent struct {
	Core             *Event //EventBody + Signature
	Round            int
	LamportTimestamp int
	Witness          bool
}

//SortedFrameEvents implements sort.Interface for []FameEvent based on
//the lamportTimestamp field.
//THIS IS A TOTAL ORDER
type SortedFrameEvents []*FrameEvent

func (a SortedFrameEvents) Len() int      { return len(a) }
func (a SortedFrameEvents) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a SortedFrameEvents) Less(i, j int) bool {
	if a[i].LamportTimestamp != a[j].LamportTimestamp {
		return a[i].LamportTimestamp < a[j].LamportTimestamp
	}

	wsi, _, _ := keys.DecodeSignature(a[i].Core.Signature)
	wsj, _, _ := keys.DecodeSignature(a[j].Core.Signature)
	return wsi.Cmp(wsj) < 0
}
