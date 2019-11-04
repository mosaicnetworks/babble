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

// EventBody contains the payload of an Event as well as the information that
// ties it to other Events. The private fields are for local computations only.
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

// Marshal returns the JSON encoding of an EventBody
func (e *EventBody) Marshal() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b) //will write to b
	if err := enc.Encode(e); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Unmarshal converts a JSON encoded EventBody to an EventBody
func (e *EventBody) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	dec := json.NewDecoder(b) //will read from b
	if err := dec.Decode(e); err != nil {
		return err
	}
	return nil
}

// Hash returns the SHA256 hash of the JSON encoded EventBody.
func (e *EventBody) Hash() ([]byte, error) {
	hashBytes, err := e.Marshal()
	if err != nil {
		return nil, err
	}
	return crypto.SHA256(hashBytes), nil
}

/*******************************************************************************
CoordinateMap
*******************************************************************************/

// EventCoordinates combines the index and hash of an Event
type EventCoordinates struct {
	Hash  string
	Index int
}

// CoordinatesMap is used by the Hashgraph consensus methods to efficiently
// calculate the StronglySee predicate.
type CoordinatesMap map[string]EventCoordinates

// NewCoordinatesMap creates an empty CoordinateMap
func NewCoordinatesMap() CoordinatesMap {
	return make(map[string]EventCoordinates)
}

// Copy creates a clone of a CoordinateMap
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

// Event is the fundamental unit of a Hashgraph. It contains an EventBody and a
// a signature of the EventBody from the Event's creator (whose public key is
// set in the EventBody.Creator byte slice). It also contains some private
// fields which are useful for local computations and ordering.
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

// NewEvent instantiates a new Event
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

// Creator returns the string representation of the creator's public key.
func (e *Event) Creator() string {
	if e.creator == "" {
		e.creator = common.EncodeToString(e.Body.Creator)
	}
	return e.creator
}

// SelfParent returns the Event's self-parent
func (e *Event) SelfParent() string {
	return e.Body.Parents[0]
}

// OtherParent returns the Event's other-parent
func (e *Event) OtherParent() string {
	return e.Body.Parents[1]
}

// Transactions returns the Event's transactions
func (e *Event) Transactions() [][]byte {
	return e.Body.Transactions
}

// InternalTransactions returns the Event's InternalTransactions
func (e *Event) InternalTransactions() []InternalTransaction {
	return e.Body.InternalTransactions
}

// Index returns the Event's index
func (e *Event) Index() int {
	return e.Body.Index
}

// BlockSignatures returns the Event's BlockSignatures
func (e *Event) BlockSignatures() []BlockSignature {
	return e.Body.BlockSignatures
}

// IsLoaded returns true if the Event contains a payload or is its creator's
// first Event.
func (e *Event) IsLoaded() bool {
	if e.Body.Index == 0 {
		return true
	}

	hasTransactions := e.Body.Transactions != nil && len(e.Body.Transactions) > 0
	hasInternalTransactions := e.Body.InternalTransactions != nil && len(e.Body.InternalTransactions) > 0

	return hasTransactions || hasInternalTransactions
}

//Sign signs the hash of the Event's body with an ecdsa sig
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

// Verify verifies the Event's signature, and the signatures of the
// InternalTransactions in its payload.
func (e *Event) Verify() (bool, error) {

	// first check signatures on internal transactions
	for _, itx := range e.Body.InternalTransactions {
		ok, err := itx.Verify()

		if err != nil {
			return false, err
		} else if !ok {
			return false, fmt.Errorf("invalid signature on internal transaction")
		}
	}

	// then check event signature
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

type eventWrapper struct {
	Body                 EventBody
	Signature            string
	CreatorID            uint32
	OtherParentCreatorID uint32
	SelfParentIndex      int
	OtherParentIndex     int
	TopologicalIndex     int
	LastAncestors        CoordinatesMap
	FirstDescendants     CoordinatesMap
}

// MarshalDB returns the JSON encoding of the Event along with some of the
// private fields which would not be exported by the default JSON marshalling
// method. These private fiels are necessary when converting to WireEvent and
// ordering in topological order. We use MarshalDB to save items to the database
// because otherwise the private fields would be lost after a write/read
// operation on the DB.
func (e *Event) MarshalDB() ([]byte, error) {

	wrapper := eventWrapper{
		Body:                 e.Body,
		Signature:            e.Signature,
		CreatorID:            e.Body.creatorID,
		OtherParentCreatorID: e.Body.otherParentCreatorID,
		SelfParentIndex:      e.Body.selfParentIndex,
		OtherParentIndex:     e.Body.otherParentIndex,
		TopologicalIndex:     e.topologicalIndex,
		LastAncestors:        e.lastAncestors,
		FirstDescendants:     e.firstDescendants,
	}

	return json.Marshal(wrapper)
}

// UnmarshalDB unmarshals a JSON encoded eventWrapper and converts it to an
// Event with some private fields set.
func (e *Event) UnmarshalDB(data []byte) error {
	var wrapper eventWrapper

	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}

	e.Body = wrapper.Body
	e.Body.creatorID = wrapper.CreatorID
	e.Body.otherParentCreatorID = wrapper.OtherParentCreatorID
	e.Body.selfParentIndex = wrapper.SelfParentIndex
	e.Body.otherParentIndex = wrapper.OtherParentIndex
	e.Signature = wrapper.Signature
	e.topologicalIndex = wrapper.TopologicalIndex
	e.lastAncestors = wrapper.LastAncestors
	e.firstDescendants = wrapper.FirstDescendants

	return nil
}

// Hash returns the SHA256 hash of the JSON-encoded body
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

// Hex returns a hex string representation of the Event's hash
func (e *Event) Hex() string {
	if e.hex == "" {
		hash, _ := e.Hash()
		e.hex = common.EncodeToString(hash)
	}

	return e.hex
}

// SetRound sets the Events round number
func (e *Event) SetRound(r int) {
	if e.round == nil {
		e.round = new(int)
	}

	*e.round = r
}

// GetRound gets the Event's round number
func (e *Event) GetRound() *int {
	return e.round
}

// SetLamportTimestamp sets the Event's Lamport Timestamp
func (e *Event) SetLamportTimestamp(t int) {
	if e.lamportTimestamp == nil {
		e.lamportTimestamp = new(int)
	}

	*e.lamportTimestamp = t
}

// SetRoundReceived sets the Event's round-received (different from round)
func (e *Event) SetRoundReceived(rr int) {
	if e.roundReceived == nil {
		e.roundReceived = new(int)
	}

	*e.roundReceived = rr
}

// SetWireInfo sets the private fields in the Event's body which are used by the
// wire representation.
func (e *Event) SetWireInfo(selfParentIndex int,
	otherParentCreatorID uint32,
	otherParentIndex int,
	creatorID uint32) {
	e.Body.selfParentIndex = selfParentIndex
	e.Body.otherParentCreatorID = otherParentCreatorID
	e.Body.otherParentIndex = otherParentIndex
	e.Body.creatorID = creatorID
}

// WireBlockSignatures returns the Event's BlockSignatures (from its payload)
// in Wire format.
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

// ToWire convert an Event to its WireEvent representation
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
WireEvent
*******************************************************************************/

// WireBody ...
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

// WireEvent ...
type WireEvent struct {
	Body      WireBody
	Signature string
}

// BlockSignatures ...
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
*******************************************************************************/

//FrameEvent is a wrapper around a regular Event. It contains exported fields
//Round, Witness, and LamportTimestamp.
type FrameEvent struct {
	Core             *Event //EventBody + Signature
	Round            int
	LamportTimestamp int
	Witness          bool
}

/*******************************************************************************
Sorting

Events and FrameEvents are sorted differently.

Events are sorted by topological order; the order in which they were inserted in
the hashgraph. Note that this order may be different for every node, so it is
not suitable for consensus ordering.

FrameEvents are sorte by LamportTimestamp where ties are broken by comparing
hashes. This is consensus total ordering.
*******************************************************************************/

// ByTopologicalOrder implements sort.Interface for []Event based on the private
// topologicalIndex field. THIS IS A PARTIAL ORDER.
type ByTopologicalOrder []*Event

// Len implements the sort.Interface
func (a ByTopologicalOrder) Len() int { return len(a) }

// Swap implements the sort.Interface
func (a ByTopologicalOrder) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less implements the sort.Interface
func (a ByTopologicalOrder) Less(i, j int) bool {
	return a[i].topologicalIndex < a[j].topologicalIndex
}

//SortedFrameEvents implements sort.Interface for []FrameEvent based on
//the lamportTimestamp field.
//THIS IS A TOTAL ORDER
type SortedFrameEvents []*FrameEvent

// Len implements the sort.Interface
func (a SortedFrameEvents) Len() int { return len(a) }

// Swap implements the sort.Interface
func (a SortedFrameEvents) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

// Less implements the sort.Interface
func (a SortedFrameEvents) Less(i, j int) bool {
	if a[i].LamportTimestamp != a[j].LamportTimestamp {
		return a[i].LamportTimestamp < a[j].LamportTimestamp
	}

	wsi, _, _ := keys.DecodeSignature(a[i].Core.Signature)
	wsj, _, _ := keys.DecodeSignature(a[j].Core.Signature)
	return wsi.Cmp(wsj) < 0
}
