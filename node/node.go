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
package node

import (
	"crypto/ecdsa"
	"sort"

	"fmt"

	"github.com/arrivets/go-swirlds/crypto"
	hg "github.com/arrivets/go-swirlds/hashgraph"
)

type Node struct {
	key *ecdsa.PrivateKey
	hg  hg.Hashgraph

	Head string
}

func NewNode(key *ecdsa.PrivateKey, participants []string) Node {
	node := Node{
		key: key,
		hg:  hg.NewHashgraph(participants),
	}
	return node
}

func (n *Node) PubKey() []byte {
	return crypto.FromECDSAPub(&n.key.PublicKey)
}

func (n *Node) GetHead() hg.Event {
	return n.hg.Events[n.Head]
}

func (n *Node) GetEvent(hash string) hg.Event {
	return n.hg.Events[hash]
}

func (n *Node) Init() error {
	initialEvent := hg.NewEvent([][]byte{},
		[]string{"", ""},
		n.PubKey())
	return n.SignAndInsertSelfEvent(initialEvent)
}

func (n *Node) SignAndInsertSelfEvent(event hg.Event) error {
	if err := event.Sign(n.key); err != nil {
		return err
	}
	if err := n.InsertEvent(event); err != nil {
		return err
	}
	n.Head = event.Hex()
	return nil
}

func (n *Node) InsertEvent(event hg.Event) error {
	return n.hg.InsertEvent(event)
}

func (n *Node) Known() map[string]int {
	return n.hg.Known()
}

//returns events that n knowns about that are not in 'known' along with n's head
func (n *Node) Diff(known map[string]int) (head string, unknown []hg.Event) {
	head = n.Head

	unknown = []hg.Event{}
	//known represents the index of last known event for every participant
	//compare this to our view of events and fill unknown with event that we known of and the other doesnt
	for p, c := range known {
		if c < len(n.hg.ParticipantEvents[p]) {
			for i := c; i < len(n.hg.ParticipantEvents[p]); i++ {
				unknown = append(unknown, n.hg.Events[n.hg.ParticipantEvents[p][i]])
			}
		}
	}
	sort.Sort(hg.ByTimestamp(unknown))

	return head, unknown
}

func (n *Node) Sync(otherHead string, unknown []hg.Event, payload [][]byte) error {
	//add unknown events
	for _, e := range unknown {
		if err := n.InsertEvent(e); err != nil {
			return err
		}
	}

	//create new event with self head and other head
	newHead := hg.NewEvent(payload,
		[]string{n.Head, otherHead},
		n.PubKey())
	if err := n.SignAndInsertSelfEvent(newHead); err != nil {
		fmt.Printf("error inserting new head")
		return err
	}

	return nil
}

func (n *Node) RunConsensus() {
	n.hg.DivideRounds()
	n.hg.DecideFame()
	n.hg.FindOrder()
}

func (n *Node) GetConsensus() []string {
	return n.hg.Consensus
}
