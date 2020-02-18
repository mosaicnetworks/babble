// +build mobile

package hashgraph

/*

This file is a duplicate of badger_store.go but imports a fork of badger db.
This fork does not attempt to acquire a directory lock as this is likely to
fail in Android 6 and below due to a bug in SELinux.

See https://github.com/mosaicnetworks/babble-android/issues/20

*/

import (
	"fmt"

	"github.com/jonknight73/badger"
	badger_options "github.com/jonknight73/badger/options"
	cm "github.com/mosaicnetworks/babble/src/common"
	"github.com/mosaicnetworks/babble/src/peers"
	"github.com/sirupsen/logrus"
)

const (
	repertoirePrefix = "rep"
	peerSetPrefix    = "peerset"
	rootSuffix       = "root"
	roundPrefix      = "round"
	topoPrefix       = "topo"
	blockPrefix      = "block"
	framePrefix      = "frame"
)

// BadgerStore contains references to the Badger database and inmem store. If
// maintenanceMode is activated, data is not written to the Badger database, but
// only to the caches.
type BadgerStore struct {
	inmemStore      *InmemStore
	db              *badger.DB
	path            string
	maintenanceMode bool
}

// NewBadgerStore opens an existing database or creates a new one if nothing is
// found in path. The maintenanceMode option deactivates writing to the
// persistant database, but adding/updating the inmem-store is preserved.
func NewBadgerStore(cacheSize int, path string, maintenanceMode bool, logger *logrus.Entry) (*BadgerStore, error) {

	opts := badger.DefaultOptions(path).
		WithSyncWrites(false).
		WithTruncate(true).
		WithTableLoadingMode(badger_options.FileIO).
		WithValueLogLoadingMode(badger_options.FileIO)

	if logger != nil {
		sub := logger.WithFields(logrus.Fields{"ns": "badger"})
		opts = opts.WithLogger(sub)
	}

	handle, err := badger.Open(opts)
	if err != nil {
		return nil, err
	}

	store := &BadgerStore{
		inmemStore:      NewInmemStore(cacheSize),
		db:              handle,
		path:            path,
		maintenanceMode: maintenanceMode,
	}
	return store, nil
}

/*******************************************************************************
Keys
*******************************************************************************/

func repertoireKey(pub string) []byte {
	return []byte(fmt.Sprintf("%s_%s", repertoirePrefix, pub))
}

func peerSetKey(round int) []byte {
	return []byte(fmt.Sprintf("%s_%09d", peerSetPrefix, round))
}

func topologicalEventKey(index int) []byte {
	return []byte(fmt.Sprintf("%s_%09d", topoPrefix, index))
}

func participantEventKey(participant string, index int) []byte {
	return []byte(fmt.Sprintf("%s__event_%09d", participant, index))
}

func participantRootKey(participant string) []byte {
	return []byte(fmt.Sprintf("%s_%s", participant, rootSuffix))
}

func roundKey(index int) []byte {
	return []byte(fmt.Sprintf("%s_%09d", roundPrefix, index))
}

func blockKey(index int) []byte {
	return []byte(fmt.Sprintf("%s_%09d", blockPrefix, index))
}

func frameKey(index int) []byte {
	return []byte(fmt.Sprintf("%s_%09d", framePrefix, index))
}

/*******************************************************************************
Implement the Store interface

BadgerStore is an implementation of the Store interface that uses an InmemStore
for caching and a BadgerDB to persist values on disk.

*******************************************************************************/

/*******************************************************************************
Cache Only

Certain objects are not meant to be retrieved directly from disk; they need to
be processed by the hashgraph methods first, and added to the InmemStore, before
they can be used. This is usually done by the Bootstrap method when a node is
started; it retrieves Events one by one from the disk, in topological order, and
inserts them in the hashgraph (thereby populating the InmemStore) before running
the consensus methods.

*******************************************************************************/

// CacheSize gets the inmem cache size
func (s *BadgerStore) CacheSize() int {
	return s.inmemStore.CacheSize()
}

// GetRound returns the round with round-number r.
func (s *BadgerStore) GetRound(r int) (*RoundInfo, error) {
	return s.inmemStore.GetRound(r)
}

// RoundWitnesses returns a round's witnesses.
func (s *BadgerStore) RoundWitnesses(r int) []string {
	round, err := s.GetRound(r)
	if err != nil {
		return []string{}
	}
	return round.Witnesses()
}

// RoundEvents returns the number of Events in round r.
func (s *BadgerStore) RoundEvents(r int) int {
	round, err := s.GetRound(r)
	if err != nil {
		return 0
	}
	return len(round.CreatedEvents)
}

// GetFrame return the Frame corresponding to round-received rr.
func (s *BadgerStore) GetFrame(rr int) (*Frame, error) {
	return s.inmemStore.GetFrame(rr)
}

// GetPeerSet returns the peer-set effective at a given round.
func (s *BadgerStore) GetPeerSet(round int) (peerSet *peers.PeerSet, err error) {
	return s.inmemStore.GetPeerSet(round)
}

// GetAllPeerSets returns the entire history of peer-sets.
func (s *BadgerStore) GetAllPeerSets() (map[int][]*peers.Peer, error) {
	return s.inmemStore.GetAllPeerSets()
}

// FirstRound returns the first round in which a given participant (identified
// by id) was a member of the corresponding peer-set.
func (s *BadgerStore) FirstRound(id uint32) (int, bool) {
	return s.inmemStore.FirstRound(id)
}

// RepertoireByPubKey returns map of peers by public-key.
func (s *BadgerStore) RepertoireByPubKey() map[string]*peers.Peer {
	return s.inmemStore.RepertoireByPubKey()
}

// RepertoireByID returns a map of peers by id.
func (s *BadgerStore) RepertoireByID() map[uint32]*peers.Peer {
	return s.inmemStore.RepertoireByID()
}

// LastEventFrom returns the hash of the last Event from a given participant.
func (s *BadgerStore) LastEventFrom(participant string) (last string, err error) {
	return s.inmemStore.LastEventFrom(participant)
}

// LastConsensusEventFrom returns the hash of the last consensus-event from a
// given participant.
func (s *BadgerStore) LastConsensusEventFrom(participant string) (last string, err error) {
	return s.inmemStore.LastConsensusEventFrom(participant)
}

// KnownEvents returns a map of participant-ID to index of last known Event.
func (s *BadgerStore) KnownEvents() map[uint32]int {
	return s.inmemStore.KnownEvents()
}

// ConsensusEvents returns the entire list of hashes of consensus-events.
func (s *BadgerStore) ConsensusEvents() []string {
	return s.inmemStore.ConsensusEvents()
}

// ConsensusEventsCount returns number of consensus events.
func (s *BadgerStore) ConsensusEventsCount() int {
	return s.inmemStore.ConsensusEventsCount()
}

// AddConsensusEvent adds a consensus event.
func (s *BadgerStore) AddConsensusEvent(event *Event) error {
	return s.inmemStore.AddConsensusEvent(event)
}

// LastRound returns the number of the last known round.
func (s *BadgerStore) LastRound() int {
	return s.inmemStore.LastRound()
}

// LastBlockIndex returns the index of the last known block.
func (s *BadgerStore) LastBlockIndex() int {
	return s.inmemStore.LastBlockIndex()
}

/*******************************************************************************
Cache + DB

The following methods use the InmemStore as a cache. When reading, values are
first fetched from the cache, and only if they are not found will they be
fetched from the BadgerDB. When writing, the value is written both to the cache
and to the DB.

*******************************************************************************/

// SetPeerSet saves a peer-set effective at a given round.
func (s *BadgerStore) SetPeerSet(round int, peerSet *peers.PeerSet) error {
	// Update the cache
	if err := s.inmemStore.SetPeerSet(round, peerSet); err != nil {
		return err
	}

	// Update the db
	if !s.maintenanceMode {
		if err := s.dbSetPeerSet(round, peerSet); err != nil {
			return err
		}
	}

	// Extend Repertoire and Roots
	for _, p := range peerSet.Peers {
		err := s.addParticipant(p)
		if err != nil {
			return err
		}
	}

	return nil
}

// addParticipant adds a participant and a corresponding Root to the database.
func (s *BadgerStore) addParticipant(p *peers.Peer) error {
	if s.maintenanceMode {
		return nil
	}

	if err := s.dbSetRepertoire(p); err != nil {
		return err
	}

	_, err := s.dbGetRoot(p.PubKeyString())
	if err != nil {
		root := NewRoot()
		if err := s.dbSetRoot(p.PubKeyString(), root); err != nil {
			return err
		}
	}

	return nil
}

// SetEvent creates or updates an Event in the store
func (s *BadgerStore) SetEvent(event *Event) error {
	// try to add it to the cache
	if err := s.inmemStore.SetEvent(event); err != nil {
		return err
	}

	// try to add it to the db
	if s.maintenanceMode {
		return nil
	}
	return s.dbSetEvents([]*Event{event})
}

// ParticipantEvents returns a participant's Event hashes, ordered by index,
// starting at index "skip".
func (s *BadgerStore) ParticipantEvents(participant string, skip int) ([]string, error) {
	res, err := s.inmemStore.ParticipantEvents(participant, skip)
	if err != nil {
		res, err = s.dbParticipantEvents(participant, skip)
	}
	return res, err
}

// ParticipantEvent returns a participant's Event for a given index.
func (s *BadgerStore) ParticipantEvent(participant string, index int) (string, error) {
	res, err := s.inmemStore.ParticipantEvent(participant, index)
	if err != nil {
		res, err = s.dbParticipantEvent(participant, index)
	}
	return res, err
}

// SetRound creates or updates a round in the store.
func (s *BadgerStore) SetRound(r int, round *RoundInfo) error {
	if err := s.inmemStore.SetRound(r, round); err != nil {
		return err
	}

	if s.maintenanceMode {
		return nil
	}
	return s.dbSetRound(r, round)
}

// GetRoot returns the Root for a given participant.
func (s *BadgerStore) GetRoot(participant string) (*Root, error) {
	root, err := s.inmemStore.GetRoot(participant)
	if err != nil {
		root, err = s.dbGetRoot(participant)
	}
	return root, mapError(err, "Root", string(participantRootKey(participant)))
}

// GetEvent returns the event identified by its hash.
func (s *BadgerStore) GetEvent(key string) (*Event, error) {
	ev, err := s.inmemStore.GetEvent(key)
	if err != nil {
		ev, err = s.dbGetEvent(key)
	}
	return ev, mapError(err, "Event", key)
}

// GetBlock returns a Block by index.
func (s *BadgerStore) GetBlock(rr int) (*Block, error) {
	res, err := s.inmemStore.GetBlock(rr)
	if err != nil {
		res, err = s.dbGetBlock(rr)
	}
	return res, mapError(err, "Block", string(blockKey(rr)))
}

// SetBlock creates or updates a Block in the Store.
func (s *BadgerStore) SetBlock(block *Block) error {
	if err := s.inmemStore.SetBlock(block); err != nil {
		return err
	}

	if s.maintenanceMode {
		return nil
	}
	return s.dbSetBlock(block)
}

// SetFrame creates or updates a Frame in the Store.
func (s *BadgerStore) SetFrame(frame *Frame) error {
	if err := s.inmemStore.SetFrame(frame); err != nil {
		return err
	}

	if s.maintenanceMode {
		return nil
	}
	return s.dbSetFrame(frame)
}

// Reset resets the Store from a given Frame.
func (s *BadgerStore) Reset(frame *Frame) error {
	// Reset InmemStore
	if err := s.inmemStore.Reset(frame); err != nil {
		return err
	}

	if s.maintenanceMode {
		return nil
	}

	// Set Frame, Roots, and PeerSet
	if err := s.dbSetFrame(frame); err != nil {
		return err
	}

	for p, root := range frame.Roots {
		if err := s.dbSetRoot(p, root); err != nil {
			return err
		}
	}

	peerSet := peers.NewPeerSet(frame.Peers)
	if err := s.dbSetPeerSet(frame.Round, peerSet); err != nil {
		return err
	}

	return nil
}

// Close closes the InmemStore and the underlying Badger database.
func (s *BadgerStore) Close() error {
	if err := s.inmemStore.Close(); err != nil {
		return err
	}
	return s.db.Close()
}

// StorePath returns the full path of the underlying Badger database directory.
func (s *BadgerStore) StorePath() string {
	return s.path
}

/*******************************************************************************
DB Methods
*******************************************************************************/

func (s *BadgerStore) dbGetRepertoire() (map[string]*peers.Peer, error) {
	repertoire := make(map[string]*peers.Peer)
	err := s.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(repertoirePrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()

			err := item.Value(func(data []byte) error {
				peer := &peers.Peer{}
				err := peer.Unmarshal(data)
				if err != nil {
					return err
				}

				repertoire[peer.PubKeyString()] = peer
				return err
			})
			//			peerBytes, err := item.Value()
			if err != nil {
				return err
			}

		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return repertoire, nil
}

func (s *BadgerStore) dbSetRepertoire(peer *peers.Peer) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()

	key := repertoireKey(peer.PubKeyString())
	val, err := peer.Marshal()
	if err != nil {
		return err
	}

	//insert [pub] => [Peer]
	if err := tx.Set(key, val); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *BadgerStore) dbGetPeerSet(round int) (*peers.PeerSet, error) {
	var peerSliceBytes []byte
	key := peerSetKey(round)
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		peerSliceBytes, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		return nil, err
	}

	return peers.NewPeerSetFromPeerSliceBytes(peerSliceBytes)
}

func (s *BadgerStore) dbSetPeerSet(round int, peerSet *peers.PeerSet) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()

	key := peerSetKey(round)
	val, err := peerSet.Marshal()
	if err != nil {
		return err
	}

	//insert [round_index] => [PeerSet bytes]
	if err := tx.Set(key, val); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *BadgerStore) dbGetEvent(key string) (*Event, error) {
	var eventBytes []byte
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}
		eventBytes, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		return nil, err
	}

	event := new(Event)
	if err := event.UnmarshalDB(eventBytes); err != nil {
		return nil, err
	}

	return event, nil
}

func (s *BadgerStore) dbSetEvents(events []*Event) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()

	for _, event := range events {
		eventHex := event.Hex()
		val, err := event.MarshalDB()
		if err != nil {
			return err
		}
		//check if it already exists
		new := false
		_, err = tx.Get([]byte(eventHex))
		if err != nil && isDBKeyNotFound(err) {
			new = true
		}
		//insert [event hash] => [event bytes]
		if err := tx.Set([]byte(eventHex), val); err != nil {
			return err
		}

		if new {
			//insert [topo_index] => [event hash]
			topoKey := topologicalEventKey(event.topologicalIndex)
			if err := tx.Set(topoKey, []byte(eventHex)); err != nil {
				return err
			}
			//insert [participant_index] => [event hash]
			peKey := participantEventKey(event.Creator(), event.Index())
			if err := tx.Set(peKey, []byte(eventHex)); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (s *BadgerStore) dbParticipantEvents(participant string, skip int) ([]string, error) {
	res := []string{}
	err := s.db.View(func(txn *badger.Txn) error {
		i := skip + 1
		key := participantEventKey(participant, i)
		item, errr := txn.Get(key)
		for errr == nil {
			v, errrr := item.ValueCopy(nil)
			if errrr != nil {
				break
			}
			res = append(res, string(v))

			i++
			key = participantEventKey(participant, i)
			item, errr = txn.Get(key)
		}

		if !isDBKeyNotFound(errr) {
			return errr
		}

		return nil
	})
	return res, err
}

func (s *BadgerStore) dbParticipantEvent(participant string, index int) (string, error) {
	data := []byte{}
	key := participantEventKey(participant, index)
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		data, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *BadgerStore) dbTopologicalEvents(start int, count int) ([]*Event, error) {
	res := []*Event{}
	t := start
	err := s.db.View(func(txn *badger.Txn) error {
		key := topologicalEventKey(t)
		item, errr := txn.Get(key)
		for errr == nil && (t < start+count) {
			v, errrr := item.ValueCopy(nil)
			if errrr != nil {
				break
			}

			evKey := string(v)
			eventItem, err := txn.Get([]byte(evKey))
			if err != nil {
				return err
			}
			eventBytes, err := eventItem.ValueCopy(nil)
			if err != nil {
				return err
			}

			event := new(Event)
			if err := event.UnmarshalDB(eventBytes); err != nil {
				return err
			}
			res = append(res, event)

			t++
			key = topologicalEventKey(t)
			item, errr = txn.Get(key)
		}

		if !isDBKeyNotFound(errr) {
			return errr
		}

		return nil
	})

	return res, err
}

func (s *BadgerStore) dbSetRoot(participant string, root *Root) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()

	key := participantRootKey(participant)

	val, err := root.Marshal()
	if err != nil {
		return err
	}

	//insert [round_index] => [round bytes]
	if err := tx.Set(key, val); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *BadgerStore) dbGetRoot(participant string) (*Root, error) {
	var rootBytes []byte
	key := participantRootKey(participant)
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		rootBytes, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		return nil, err
	}

	root := new(Root)
	if err := root.Unmarshal(rootBytes); err != nil {
		return nil, err
	}

	return root, nil
}

func (s *BadgerStore) dbGetRound(index int) (*RoundInfo, error) {
	var roundBytes []byte
	key := roundKey(index)
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		roundBytes, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		return nil, err
	}

	roundInfo := new(RoundInfo)
	if err := roundInfo.Unmarshal(roundBytes); err != nil {
		return nil, err
	}

	return roundInfo, nil
}

func (s *BadgerStore) dbSetRound(index int, round *RoundInfo) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()

	key := roundKey(index)
	val, err := round.Marshal()
	if err != nil {
		return err
	}

	//insert [round_index] => [round bytes]
	if err := tx.Set(key, val); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *BadgerStore) dbGetBlock(index int) (*Block, error) {
	var blockBytes []byte
	key := blockKey(index)
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		blockBytes, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		return nil, err
	}

	block := new(Block)
	if err := block.Unmarshal(blockBytes); err != nil {
		return nil, err
	}

	return block, nil
}

func (s *BadgerStore) dbSetBlock(block *Block) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()

	key := blockKey(block.Index())
	val, err := block.Marshal()
	if err != nil {
		return err
	}

	//insert [index] => [block bytes]
	if err := tx.Set(key, val); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *BadgerStore) dbGetFrame(index int) (*Frame, error) {
	var frameBytes []byte
	key := frameKey(index)
	err := s.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}
		frameBytes, err = item.ValueCopy(nil)
		return err
	})

	if err != nil {
		return nil, err
	}

	frame := new(Frame)
	if err := frame.Unmarshal(frameBytes); err != nil {
		return nil, err
	}

	return frame, nil
}

func (s *BadgerStore) dbSetFrame(frame *Frame) error {
	tx := s.db.NewTransaction(true)
	defer tx.Discard()

	key := frameKey(frame.Round)
	val, err := frame.Marshal()
	if err != nil {
		return err
	}

	//insert [index] => [block bytes]
	if err := tx.Set(key, val); err != nil {
		return err
	}

	return tx.Commit()
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

func isDBKeyNotFound(err error) bool {
	return err != nil && err.Error() == badger.ErrKeyNotFound.Error()
}

func mapError(err error, name, key string) error {
	if err != nil {
		if isDBKeyNotFound(err) {
			return cm.NewStoreErr(name, cm.KeyNotFound, key)
		}
	}
	return err
}

//GetMaintenanceMode is a getter
func (s *BadgerStore) GetMaintenanceMode() bool {
	return s.maintenanceMode
}

//SetMaintenanceMode is a setter
func (s *BadgerStore) SetMaintenanceMode(val bool) {
	s.maintenanceMode = val
}
