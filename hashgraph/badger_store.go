package hashgraph

import (
	"fmt"

	cm "github.com/babbleio/babble/common"
	"github.com/dgraph-io/badger"
)

type BadgerStore struct {
	inmemStore *InmemStore
	db         *badger.DB
	path       string
}

func NewBadgerStore(participants map[string]int, cacheSize int, path string) (*BadgerStore, error) {
	inmemStore := NewInmemStore(participants, cacheSize)
	opts := badger.DefaultOptions
	opts.Dir = path
	opts.ValueDir = path
	opts.SyncWrites = false
	handle, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}
	store := &BadgerStore{
		inmemStore: inmemStore,
		db:         handle,
		path:       path,
	}
	if err := store.setRootsToDB(inmemStore.roots); err != nil {
		return nil, err
	}
	return store, nil
}

//==============================================================================
//Keys

func participantEventKey(participant string, index int) []byte {
	return []byte(fmt.Sprintf("%s%09d", participant, index))
}

func participantRootKey(participant string) []byte {
	return []byte(fmt.Sprintf("%s_root", participant))
}

//==============================================================================
//Implement the Store interface

func (s *BadgerStore) CacheSize() int {
	return s.inmemStore.CacheSize()
}

func (s *BadgerStore) GetEvent(key string) (event Event, err error) {
	//try to get it from cache
	event, err = s.inmemStore.GetEvent(key)
	//try to get it from db
	if err != nil {
		event, err = s.getEventFromDB(key)
	}
	return event, err
}

func (s *BadgerStore) SetEvent(event Event) error {
	//try to add it to the cache
	if err := s.inmemStore.SetEvent(event); err != nil {
		return err
	}
	//try to add it to the db
	if _, err := s.getEventFromDB(event.Hex()); err != nil {
		return s.setEventsToDB([]Event{event})
	}
	return nil
}

func (s *BadgerStore) ParticipantEvents(participant string, skip int) ([]string, error) {
	res, err := s.inmemStore.ParticipantEvents(participant, skip)
	if err != nil {
		res, err = s.participantEventsFromDB(participant, skip)
	}
	return res, err
}

func (s *BadgerStore) ParticipantEvent(participant string, index int) (string, error) {
	result, err := s.inmemStore.ParticipantEvent(participant, index)
	if err != nil {
		result, err = s.participantEventFromDB(participant, index)
	}
	return result, err
}

func (s *BadgerStore) LastFrom(participant string) (last string, isRoot bool, err error) {
	return s.inmemStore.LastFrom(participant)
}

func (s *BadgerStore) Known() map[int]int {
	return s.inmemStore.Known()
}

func (s *BadgerStore) ConsensusEvents() []string {
	return s.inmemStore.ConsensusEvents()
}

func (s *BadgerStore) ConsensusEventsCount() int {
	return s.inmemStore.ConsensusEventsCount()
}

func (s *BadgerStore) AddConsensusEvent(key string) error {
	return s.inmemStore.AddConsensusEvent(key)
}

func (s *BadgerStore) GetRound(r int) (RoundInfo, error) {
	return s.inmemStore.GetRound(r)
}

func (s *BadgerStore) SetRound(r int, round RoundInfo) error {
	return s.inmemStore.SetRound(r, round)
}

func (s *BadgerStore) LastRound() int {
	return s.inmemStore.LastRound()
}

func (s *BadgerStore) Rounds() int {
	return s.inmemStore.Rounds()
}

func (s *BadgerStore) RoundWitnesses(r int) []string {
	return s.inmemStore.RoundWitnesses(r)
}

func (s *BadgerStore) RoundEvents(r int) int {
	return s.inmemStore.RoundEvents(r)
}

func (s *BadgerStore) GetRoot(participant string) (Root, error) {
	root, err := s.inmemStore.GetRoot(participant)
	if err != nil {
		root, err = s.getRootFromDB(participant)
	}
	return root, err
}

func (s *BadgerStore) Reset(roots map[string]Root) error {
	return s.inmemStore.Reset(roots)
}

func (s *BadgerStore) Close() error {
	if err := s.inmemStore.Close(); err != nil {
		return err
	}
	return s.db.Close()
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//DB Methods

func (s *BadgerStore) getEventFromDB(key string) (Event, error) {
	var eventBytes []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		eventBytes, err = item.Value()
		return err
	})

	if err != nil {
		return Event{}, cm.NewStoreErr(cm.KeyNotFound, key)
	}

	event := new(Event)
	if err := event.Unmarshal(eventBytes); err != nil {
		return Event{}, err
	}

	return *event, nil
}

func (s *BadgerStore) setEventsToDB(events []Event) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()
	for _, event := range events {
		eventHex := event.Hex()
		val, err := event.Marshal()
		if err != nil {
			return err
		}
		//insert [event hash] => [event bytes]
		if err := tx.Set([]byte(eventHex), val); err != nil {
			return err
		}
		//insert [(participant,index)] => [event hash]
		peKey := participantEventKey(event.Creator(), event.Index())
		if err := tx.Set(peKey, []byte(eventHex)); err != nil {
			return err
		}

	}
	return tx.Commit(nil)
}

func (s *BadgerStore) participantEventsFromDB(participant string, skip int) ([]string, error) {
	res := []string{}
	err := s.db.View(func(txn *badger.Txn) error {
		i := skip + 1
		key := participantEventKey(participant, i)
		item, errr := txn.Get(key)
		for errr == nil {
			v, errrr := item.Value()
			if errrr != nil {
				break
			}
			res = append(res, string(v))

			i++
			key = participantEventKey(participant, i)
			item, errr = txn.Get(key)
		}

		if errr.Error() != badger.ErrKeyNotFound.Error() {
			return errr
		}

		return nil
	})
	return res, err
}

func (s *BadgerStore) participantEventFromDB(participant string, index int) (string, error) {
	data := []byte{}
	key := participantEventKey(participant, index)
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		data, err = item.Value()
		return err
	})
	if err != nil {
		return "", cm.NewStoreErr(cm.KeyNotFound, fmt.Sprintf("(%s,%d)", participant, index))
	}
	return string(data), nil
}

func (s *BadgerStore) setRootsToDB(roots map[string]Root) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()
	for participant, root := range roots {
		val, err := root.Marshal()
		if err != nil {
			return err
		}
		key := participantRootKey(participant)
		//insert [participant_root] => [root bytes]
		if err := tx.Set(key, val); err != nil {
			return err
		}
	}
	return tx.Commit(nil)
}

func (s *BadgerStore) getRootFromDB(participant string) (Root, error) {
	var rootBytes []byte
	key := participantRootKey(participant)
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		rootBytes, err = item.Value()
		return err
	})

	if err != nil {
		return Root{}, cm.NewStoreErr(cm.KeyNotFound, fmt.Sprintf("%s_root", participant))
	}

	root := new(Root)
	if err := root.Unmarshal(rootBytes); err != nil {
		return Root{}, err
	}

	return *root, nil
}
