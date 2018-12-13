package hashgraph

import (
	"bytes"
	"fmt"

	"github.com/ugorji/go/codec"
)

/*
Roots constitute the base of a Hashgraph. Each Participant is assigned a Root on
top of which Events will be added. The first Event of a participant must have a
Self-Parent and an Other-Parent that match its Root X and Y respectively.

This construction allows us to initialize Hashgraphs where the first Events are
taken from the middle of another Hashgraph

ex 1:

-----------------        -----------------       -----------------
- Event E0      -        - Event E1      -       - Event E2      -
- SP = ""       -        - SP = ""       -       - SP = ""       -
- OP = ""       -        - OP = ""       -       - OP = ""       -
-----------------        -----------------       -----------------
        |                        |                       |
-----------------		 -----------------		 -----------------
- Root 0        - 		 - Root 1        - 		 - Root 2        -
- X = Y = ""    - 		 - X = Y = ""    -		 - X = Y = ""    -
- Index= -1     -		 - Index= -1     -       - Index= -1     -
- Others= empty - 		 - Others= empty -       - Others= empty -
-----------------		 -----------------       -----------------

ex 2:

-----------------
- Event E02     -
- SP = E01      -
- OP = E_OLD    -
-----------------
       |
-----------------
- Event E01     -
- SP = E00      -
- OP = E10      -  \
-----------------    \
       |               \
-----------------        -----------------       -----------------
- Event E00     -        - Event E10     -       - Event E20     -
- SP = x0       -        - SP = x1       -       - SP = x2       -
- OP = y0       -        - OP = y1       -       - OP = y2       -
-----------------        -----------------       -----------------
        |                        |                       |
-----------------		 -----------------		 -----------------
- Root 0        - 		 - Root 1        - 		 - Root 2        -
- X: x0, Y: y0  - 		 - X: x1, Y: y1  - 		 - X: x2, Y: y2  -
- Index= i0     -		 - Index= i1     -       - Index= i2     -
- Others= {     - 		 - Others= empty -       - Others= empty -
-  E02: E_OLD   -        -----------------       -----------------
- }             -
-----------------
*/

//RootEvent contains enough information about an Event to insert its direct
//descendant in a reset Hashgraph.
type RootEvent struct {
	Hash      string
	CreatorID uint32
	Index     int
}

//NewBaseRootEvent creates a RootEvent corresponding to the the very beginning
//of a Hashgraph.
func NewBaseRootEvent(creatorID uint32) RootEvent {
	res := RootEvent{
		Hash:      fmt.Sprintf("Root%d", creatorID),
		CreatorID: creatorID,
		Index:     -1,
	}
	return res
}

/*
RoundAndLamport holds information about an Event's Round and LamportTimestamp.
It is used to store the pre-computed values for Events in a Root, so when
resetting a hashgraph from a Frame, these values do not need to be
re-calculated.
*/
type RoundAndLamport struct {
	Round            int
	LamportTimestamp int
}

//NewBaseRoundAndLamport creates a new RoundAndLamport for a fresh Root.
//XXX would be nice to start all indexes at 0
func NewBaseRoundAndLamport() RoundAndLamport {
	return RoundAndLamport{
		Round:            -1,
		LamportTimestamp: -1,
	}
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
	BaseEvent        RootEvent
	BasePrecomputed  RoundAndLamport
	OtherEvents      map[string]RootEvent
	OtherPrecomputed map[string]RoundAndLamport
}

//NewBaseRoot initializes a Root object for a fresh Hashgraph.
func NewBaseRoot(creatorID uint32) *Root {
	res := &Root{
		BaseEvent:        NewBaseRootEvent(creatorID),
		BasePrecomputed:  NewBaseRoundAndLamport(),
		OtherEvents:      make(map[string]RootEvent),
		OtherPrecomputed: make(map[string]RoundAndLamport),
	}
	return res
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

func (root *Root) Unmarshal(data []byte) error {
	b := bytes.NewBuffer(data)
	jh := new(codec.JsonHandle)
	jh.Canonical = true
	dec := codec.NewDecoder(b, jh)

	return dec.Decode(root)
}
