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
	"encoding/binary"

	"github.com/boltdb/bolt"
)

var (
	eventBkt     = []byte("events")
	consensusBkt = []byte("consensus")
	roundBkt     = []byte("rounds")
)

type BoltStore struct {
	fn           string
	db           *bolt.DB
	participants []string
}

func NewBoltStore(fn string, participants []string) (*BoltStore, error) {
	db, err := bolt.Open(fn, 0600, nil)
	if err != nil {
		return nil, err
	}
	s := &BoltStore{
		fn:           fn,
		db:           db,
		participants: participants,
	}
	err = s.init()
	if err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}

func (s *BoltStore) init() error {
	tx, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Create all the buckets
	if _, err := tx.CreateBucketIfNotExists(eventBkt); err != nil {
		return err
	}
	if _, err := tx.CreateBucketIfNotExists(consensusBkt); err != nil {
		return err
	}
	for _, p := range s.participants {
		_, err = tx.CreateBucketIfNotExists([]byte(p))
		if err != nil {
			return err
		}
	}
	if _, err := tx.CreateBucketIfNotExists(roundBkt); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *BoltStore) Close() error {
	return s.db.Close()
}

func (s *BoltStore) GetEvent(hash string) (Event, error) {
	var event Event

	var data []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket(eventBkt)
		data = b.Get([]byte(hash))
		if data == nil {
			return ErrKeyNotFound
		}
		return nil
	})
	if err != nil {
		return event, err
	}

	err = event.Unmarshal(data)

	return event, err
}

func (s *BoltStore) SetEvent(event Event) error {
	data, err := event.Marshal()
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(eventBkt)
		if err := b.Put([]byte(event.Hex()), data); err != nil {
			return err
		}

		c := tx.Bucket([]byte(event.Creator()))
		id, _ := c.NextSequence()
		return c.Put(itob(int(id)), []byte(event.Hex()))
	})
}

func (s *BoltStore) ParticipantEvents(participant string) ([]string, error) {
	events := []string{}
	err := s.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(participant))
		if b == nil {
			return ErrKeyNotFound
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			events = append(events, string(v))
		}
		return nil
	})
	return events, err
}

func (s *BoltStore) Known() map[string]int {
	known := make(map[string]int)
	s.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		for _, p := range s.participants {
			count := 0
			b := tx.Bucket([]byte(p))
			if b == nil {
				return ErrKeyNotFound
			}
			c := b.Cursor()
			if last, _ := c.Last(); last != nil {
				count = int(binary.BigEndian.Uint64(last))
			}
			known[p] = count
		}
		return nil
	})
	return known
}

func (s *BoltStore) ConsensusEvents() []string {
	events := []string{}
	s.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket(consensusBkt)
		if b == nil {
			return ErrKeyNotFound
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			events = append(events, string(v))
		}
		return nil
	})
	return events
}

func (s *BoltStore) AddConsensusEvent(key string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(consensusBkt)
		id, _ := b.NextSequence()
		return b.Put(itob(int(id)), []byte(key))
	})
}

func (s *BoltStore) GetRound(r int) (RoundInfo, error) {
	var round RoundInfo

	var data []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket(roundBkt)
		data = b.Get(itob(r))
		if data == nil {
			return ErrKeyNotFound
		}
		return nil
	})
	if err != nil {
		return round, err
	}

	err = round.Unmarshal(data)

	return round, err
}

func (s *BoltStore) SetRound(r int, round RoundInfo) error {
	data, err := round.Marshal()
	if err != nil {
		return err
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(roundBkt)
		return b.Put(itob(r), data)
	})
}

func (s *BoltStore) Rounds() int {
	rounds := 0
	s.db.View(func(tx *bolt.Tx) error {

		b := tx.Bucket(roundBkt)
		if b == nil {
			return ErrKeyNotFound
		}
		c := b.Cursor()
		if last, _ := c.Last(); last != nil {
			rounds = int(binary.BigEndian.Uint64(last)) + 1
		}

		return nil
	})
	return rounds
}

func (s *BoltStore) RoundWitnesses(r int) []string {
	round, err := s.GetRound(r)
	if err != nil {
		return []string{}
	}
	return round.Witnesses()
}

func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}
