package hashgraph

import (
	"bytes"

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

//RootEvent contains enough information about an Event and its direct descendant
//to allow inserting Events on top of it.
type RootEvent struct {
	Hash                            string
	LamportTimestamp                int
	Round                           int
	DescendantStronglySeenWitnesses int  //How many witnesses of Round the direct descendant of this Event can StronglySee
	DescendantWitness               bool //Is the direct descendant a witness?
}

//NewBaseRootEvent creates a RootEvent corresponding to the the very beginning
//of a Hashgraph.
func NewBaseRootEvent() RootEvent {
	return RootEvent{
		LamportTimestamp:  -1,
		Round:             -1,
		DescendantWitness: true,
	}
}

//Root forms a base on top of which a participant's Events can be inserted. In
//contains the parents of the first descendant of the Root (X and Y), as well as
//other Events, belonging to a past before the Root, which might be referenced
//in future Events.
type Root struct {
	X, Y   RootEvent //SelfParent and OtherParent
	Index  int       //Index of SelfParent
	Others map[string]RootEvent
}

//NewBaseRoot initializes a Root object for a fresh Hashgraph.
func NewBaseRoot() Root {
	return Root{
		X:      NewBaseRootEvent(),
		Y:      NewBaseRootEvent(),
		Index:  -1,
		Others: map[string]RootEvent{},
	}
}

//The JSON encoding of a Root must be DETERMINISTIC because it is itself
//included in the JSON encoding of a Frame. The difficulty is that Roots contain
//go maps for which one should not expect a de facto order of entries; we cannot
//use the builtin JSON codec within overriding something. Instead, we are using
//a third party library (ugorji/codec) that enables deterministic encoding of
//golang maps.
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
