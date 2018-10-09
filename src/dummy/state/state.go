package state

import (
	"fmt"
	"os"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/mosaicnetworks/babble/src/hashgraph"
	"github.com/sirupsen/logrus"
)

/*
* The dummy App is used for testing and as an example for building Babble
* applications. Here, we define the dummy's state which doesn't really do
* anything useful. It writes block transactions (as strings) into a messages.txt
* file. The state hash is computed by hashing transactions together. Snapshots
* correspond to the state hash resulting from executing a the block's
* transactions.
 */

type State struct {
	stateHash []byte
	snapshots map[int][]byte
	logger    *logrus.Logger
}

func NewState(logger *logrus.Logger) *State {
	state := &State{
		stateHash: []byte{},
		snapshots: make(map[int][]byte),
		logger:    logger,
	}
	state.writeMessage([]byte("init"))
	return state
}

func (a *State) CommitBlock(block hashgraph.Block) ([]byte, error) {
	a.logger.WithField("block", block).Debug("CommitBlock")
	err := a.writeBlock(block)
	if err != nil {
		return nil, err
	}
	return a.stateHash, nil
}

func (a *State) GetSnapshot(blockIndex int) ([]byte, error) {
	a.logger.WithField("block", blockIndex).Debug("GetSnapshot")

	snapshot, ok := a.snapshots[blockIndex]
	if !ok {
		return nil, fmt.Errorf("Snapshot %d not found", blockIndex)
	}

	return snapshot, nil
}

func (a *State) Restore(snapshot []byte) ([]byte, error) {
	//XXX do something smart here
	a.stateHash = snapshot
	return a.stateHash, nil
}

func (a *State) writeBlock(block hashgraph.Block) error {
	file, err := a.getFile()
	if err != nil {
		a.logger.Error(err)
		return err
	}
	defer file.Close()

	// write some text to file
	//and update state hash
	hash := a.stateHash
	for _, tx := range block.Transactions() {
		_, err = file.WriteString(fmt.Sprintf("%s\n", string(tx)))
		if err != nil {
			a.logger.Error(err)
			return err
		}
		hash = crypto.SimpleHashFromTwoHashes(hash, crypto.SHA256(tx))
	}

	err = file.Sync()
	if err != nil {
		a.logger.Error(err)
		return err
	}

	a.stateHash = hash

	//XXX do something smart here
	a.snapshots[block.Index()] = hash

	return nil
}

func (a *State) writeMessage(tx []byte) {
	file, err := a.getFile()
	if err != nil {
		a.logger.Error(err)
		return
	}
	defer file.Close()

	// write some text to file
	_, err = file.WriteString(fmt.Sprintf("%s\n", string(tx)))
	if err != nil {
		a.logger.Error(err)
	}
	err = file.Sync()
	if err != nil {
		a.logger.Error(err)
	}
}

func (a *State) getFile() (*os.File, error) {
	path := "messages.txt"
	return os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
}
