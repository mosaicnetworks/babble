package hashgraph

import "github.com/dgraph-io/badger"

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
	return store, nil
}

//==============================================================================
//Implement the Store interface

func (s *BadgerStore) CacheSize() int {
	return s.inmemStore.CacheSize()
}

func (s *BadgerStore) GetEvent(key string) (Event, error) {
	return s.inmemStore.GetEvent(key)
}

func (s *BadgerStore) SetEvent(event Event) error {
	return s.inmemStore.SetEvent(event)
}

func (s *BadgerStore) ParticipantEvents(participant string, skip int) ([]string, error) {
	return s.inmemStore.ParticipantEvents(participant, skip)
}

func (s *BadgerStore) ParticipantEvent(participant string, index int) (string, error) {
	return s.inmemStore.ParticipantEvent(participant, index)
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
	return s.inmemStore.GetRoot(participant)
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
