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
	"fmt"
	"sort"

	"github.com/arrivets/go-swirlds/crypto"
	hg "github.com/arrivets/go-swirlds/hashgraph"
)

type Core struct {
	key *ecdsa.PrivateKey
	hg  hg.Hashgraph

	Head string
}

func NewCore(key *ecdsa.PrivateKey, participants []string) Core {
	core := Core{
		key: key,
		hg:  hg.NewHashgraph(participants),
	}
	return core
}

func (c *Core) PubKey() []byte {
	return crypto.FromECDSAPub(&c.key.PublicKey)
}

func (c *Core) GetHead() hg.Event {
	return c.hg.Events[c.Head]
}

func (c *Core) GetEvent(hash string) hg.Event {
	return c.hg.Events[hash]
}

func (c *Core) Init() error {
	initialEvent := hg.NewEvent([][]byte{},
		[]string{"", ""},
		c.PubKey())
	return c.SignAndInsertSelfEvent(initialEvent)
}

func (c *Core) SignAndInsertSelfEvent(event hg.Event) error {
	if err := event.Sign(c.key); err != nil {
		return err
	}
	if err := c.InsertEvent(event); err != nil {
		return err
	}
	c.Head = event.Hex()
	return nil
}

func (c *Core) InsertEvent(event hg.Event) error {
	return c.hg.InsertEvent(event)
}

func (c *Core) Known() map[string]int {
	return c.hg.Known()
}

//returns events that n knowns about that are not in 'known' along with n's head
func (c *Core) Diff(known map[string]int) (head string, unknown []hg.Event) {
	head = c.Head

	unknown = []hg.Event{}
	//known represents the index of last known event for every participant
	//compare this to our view of events and fill unknown with event that we known of and the other doesnt
	for p, ct := range known {
		if ct < len(c.hg.ParticipantEvents[p]) {
			for i := ct; i < len(c.hg.ParticipantEvents[p]); i++ {
				unknown = append(unknown, c.hg.Events[c.hg.ParticipantEvents[p][i]])
			}
		}
	}
	sort.Sort(hg.ByTimestamp(unknown))

	return head, unknown
}

func (c *Core) Sync(otherHead string, unknown []hg.Event, payload [][]byte) error {
	//add unknown events
	for _, e := range unknown {
		if err := c.InsertEvent(e); err != nil {
			return err
		}
	}

	//create new event with self head and other head
	newHead := hg.NewEvent(payload,
		[]string{c.Head, otherHead},
		c.PubKey())
	if err := c.SignAndInsertSelfEvent(newHead); err != nil {
		fmt.Printf("error inserting new head")
		return err
	}

	return nil
}

func (c *Core) RunConsensus() {
	c.hg.DivideRounds()
	c.hg.DecideFame()
	c.hg.FindOrder()
}

func (c *Core) GetConsensus() []string {
	return c.hg.Consensus
}
