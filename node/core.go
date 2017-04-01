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

	"bitbucket.org/mosaicnet/babble/crypto"
	hg "bitbucket.org/mosaicnet/babble/hashgraph"
)

type Core struct {
	key *ecdsa.PrivateKey
	hg  hg.Hashgraph

	Head string
}

func NewCore(key *ecdsa.PrivateKey, participants []string, store hg.Store, commitCh chan []hg.Event) Core {
	core := Core{
		key: key,
		hg:  hg.NewHashgraph(participants, store, commitCh),
	}
	return core
}

func (c *Core) PubKey() []byte {
	return crypto.FromECDSAPub(&c.key.PublicKey)
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

//returns events that c knowns about that are not in 'known', along with c's head
func (c *Core) Diff(known map[string]int) (head string, unknown []hg.Event, err error) {
	head = c.Head

	unknown = []hg.Event{}
	//known represents the number of events known for every participant
	//compare this to our view of events and fill unknown with events that we know of
	// and the other doesnt
	for p, ct := range known {
		participantEvents, err := c.hg.Store.ParticipantEvents(p, ct)
		if err != nil {
			return "", []hg.Event{}, err
		}
		for _, e := range participantEvents {
			ev, err := c.hg.Store.GetEvent(e)
			if err != nil {
				return "", []hg.Event{}, err
			}
			unknown = append(unknown, ev)
		}
	}
	sort.Sort(hg.ByTimestamp(unknown))

	return head, unknown, nil
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
		return fmt.Errorf("Error inserting new head: %s", err)
	}

	return nil
}

func (c *Core) RunConsensus() error {
	if err := c.hg.DivideRounds(); err != nil {
		return err
	}
	if err := c.hg.DecideFame(); err != nil {
		return err
	}
	if err := c.hg.FindOrder(); err != nil {
		return err
	}
	return nil
}

func (c *Core) GetHead() (hg.Event, error) {
	return c.hg.Store.GetEvent(c.Head)
}

func (c *Core) GetEvent(hash string) (hg.Event, error) {
	return c.hg.Store.GetEvent(hash)
}

func (c *Core) GetEventTransactions(hash string) ([][]byte, error) {
	var txs [][]byte
	ex, err := c.GetEvent(hash)
	if err != nil {
		return txs, err
	}
	txs = ex.Transactions()
	return txs, nil
}

func (c *Core) GetConsensusEvents() []string {
	return c.hg.ConsensusEvents()
}

func (c *Core) GetConsensusEventsCount() int {
	return c.hg.Store.ConsensusEventsCount()
}

func (c *Core) GetUndeterminedEvents() []string {
	return c.hg.UndeterminedEvents
}

func (c *Core) GetConsensusTransactions() ([][]byte, error) {
	txs := [][]byte{}
	for _, e := range c.GetConsensusEvents() {
		eTxs, err := c.GetEventTransactions(e)
		if err != nil {
			return txs, fmt.Errorf("Consensus event not found: %s", e)
		}
		txs = append(txs, eTxs...)
	}
	return txs, nil
}

func (c *Core) GetLastConsensusRoundIndex() *int {
	return c.hg.LastConsensusRound
}

func (c *Core) GetConsensusTransactionsCount() int {
	return c.hg.ConsensusTransactions
}
