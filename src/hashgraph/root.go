package hashgraph

import (
	"bytes"
	"fmt"

	"github.com/ugorji/go/codec"
)

/*
Roots constitute the base of a Hashgraph. Each Participant is assigned a Root on
top of which Events will be added. The first Event of a participant must have a
Self-Parent that matches its Root Head. The OtherParents can be looked up in the
Roots of other participants. Roots contain a set of Past Events, the depth of
which is hard code as ROOT_DEPTH for now, that form a secondary store (on top of
the hg Store) when the Hashgraph is Reset/Pruned. The Root also contains
precomputed  Rounds and LamportTimestamps for consensus Events belonging to the
Frame above the Root; this facilitates consensus computations on a Reset
hashgraph. Roots are a very important component of Frames, which are standalone
sections of the Hasghraph.

ex 1:

 ---------------          ---------------         ---------------
| Event E0      |        | Event E1      |       | Event E2      |
| SP = "R0"     |        | SP = "R1"     |       | SP = "R2"     |
| OP = ""       |        | OP = ""       |       | OP = ""       |
 ---------------          ---------------         ---------------
        |                        |                       |
 ---------------          ---------------         ---------------
| Root 0        |        | Root 1        |       | Root 2        |
| Head=R0       |        | Head=R1       |       | Head=R2       |
| Past={        |        | Past={        |       | Past={        |
|   R0: {--},   |        |  R1: {--},    |       |  R2: {--},    |
| }             |        | }             |       | }             |
| Precomputed={}|        | Precomputed={}|       | Precomputed={}|
 ---------------          ---------------         ---------------

ex 2:

 ---------------
| Event E02     |
| SP = E01      |
| OP = E_OLD    |
 ---------------
       |
 ---------------
| Event E01     |
| SP = E00      |
| OP = E10      |  \
 ---------------     \
       |               \
 ---------------          ---------------         ---------------
| Event E00     |        | Event E10     |       | Event E20     |
| SP = x0       |        | SP = x1       |       | SP = x2       |
| OP = y0       |        | OP = y1       |       | OP = y2       |
 ---------------          ---------------         ---------------
        |                        |                       |
 ---------------          ---------------         ---------------
| Root 0        |        | Root 1        |       | Root 2        |
| Head = x0     |        | Head = x1     |       | Head = x2     |
| Past={        |        | Past={        |       | Past={        |
|   x0: {--}    |        |   x1: {--}    |       |   x2: {--}    |
|   y2: {--}    |        |   y0: {--}    |       |   y1: {--}    |
| }             |        | }             |       | }             |
| Precomputed={ |        | Precomputed={ |       | Precomputed={ |
|  E00: {--}    |        |    E10: {--}  |       |   E20: {--}   |
|  E01: {--}    |        | }             |       | }             |
|  E02: {--}    |         ---------------         ---------------
| }             |
 ---------------
*/

//rootSelfParent is the name we give to base Roots, what the first Event's self-
//parent should be.
func rootSelfParent(participantID uint32) string {
	return fmt.Sprintf("Root%d", participantID)
}

//RootEvent contains enough information about an Event to insert its direct
//descendant in a reset Hashgraph.
type RootEvent struct {
	Hash             string
	CreatorID        uint32
	Index            int
	Round            int
	LamportTimestamp int
}

//NewBaseRootEvent creates a RootEvent corresponding to the the very beginning
//of a Hashgraph.
func NewBaseRootEvent(creatorID uint32) RootEvent {
	res := RootEvent{
		Hash:             rootSelfParent(creatorID),
		CreatorID:        creatorID,
		Index:            -1,
		Round:            -1,
		LamportTimestamp: -1,
	}
	return res
}

/*
Root forms a base on top of which a participant's Events can be inserted. It
contains the SelfParent of the first descendant of the Root, the pre-computed
Round and Lamport values of Events attached to the Root (cf Frame), as well as
other Events, that are referenced by future Events but are "below" the Frame.
The Others field is necessary when the other-parent of a Frame Event is not
contained in the Frame, because it is not permitted to insert an Event in the
Hashgraph if both its parents are not known.
*/
type Root struct {
	Head        string
	Past        map[string]RootEvent
	Precomputed map[string]RootEvent

	pastByIndex map[int]string //Index => Hash
}

//NewRoot initializes a new Root with a head.
func NewRoot(head RootEvent) *Root {
	res := &Root{
		Head:        head.Hash,
		Past:        make(map[string]RootEvent),
		Precomputed: make(map[string]RootEvent),
		pastByIndex: make(map[int]string),
	}
	res.Insert(head)
	return res
}

//NewBaseRoot initializes a Root object for a fresh Hashgraph.
func NewBaseRoot(creatorID uint32) *Root {
	head := NewBaseRootEvent(creatorID)
	res := NewRoot(head)
	return res
}

//Init populates the private lookup maps
func (root *Root) Init() {
	root.pastByIndex = make(map[int]string)
	for i, re := range root.Past {
		root.pastByIndex[re.Index] = i
	}
}

//Insert adds a RootEvent to Past and updates lookup maps
func (root *Root) Insert(re RootEvent) {
	root.Past[re.Hash] = re
	root.pastByIndex[re.Index] = re.Hash
}

//GetHead returns the Root's top RootEvent
func (root *Root) GetHead() RootEvent {
	return root.Past[root.Head]
}

//PastByIndex attempts to retrieve a Past Event given its index instead of its
//Hash.
func (root *Root) PastByIndex(index int) (RootEvent, bool) {
	if root.pastByIndex == nil {
		root.Init()
	}

	h, ok := root.pastByIndex[index]
	if !ok {
		return RootEvent{}, false
	}

	res, ok := root.Past[h]

	return res, ok
}

/*
The JSON encoding of a Root must be DETERMINISTIC because it is itself included
in the JSON encoding of a Frame. The difficulty is that Roots contain go maps
for which one should not expect a de facto order of entries; so we cannot use
the builtin JSON codec. Instead, we are using a third party library
(ugorji/codec) that enables deterministic encoding of golang maps.
*/
func (root *Root) Marshal() ([]byte, error) {
	b := new(bytes.Buffer)
	jh := new(codec.JsonHandle)
	jh.Canonical = true
	enc := codec.NewEncoder(b, jh)

	if err := enc.Encode(root); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

//Unmarshal decodes data into a Root object and calls Init to populate the
//private lookup maps
func (root *Root) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	jh := new(codec.JsonHandle)
	jh.Canonical = true
	dec := codec.NewDecoder(b, jh)

	return dec.Decode(root)
}
