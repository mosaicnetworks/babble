package common

import "fmt"

// StoreErrType encodes the nature of a StoreErr
type StoreErrType uint32

const (
	// KeyNotFound signifies an item is not found.
	KeyNotFound StoreErrType = iota
	// TooLate signifies that an item is no longer in the store because it was
	// evicted from the cache.
	TooLate
	// SkippedIndex signifies that an attempt was made to insert an
	// non-sequential item in a cache.
	SkippedIndex
	// UnknownParticipant signifies that an attempt was made to retrieve objects
	// associated to a non-existant participant.
	UnknownParticipant
	// Empty signifies that a cache is empty.
	Empty
	// KeyAlreadyExists Signifies that an attempt was made to insert an item
	// already present in a cache.
	KeyAlreadyExists
)

// StoreErr is a generic error type that encodes errors when accessing objects
// in the hashgraph store.
type StoreErr struct {
	dataType string
	errType  StoreErrType
	key      string
}

// NewStoreErr creates a StoreErr pertaining to an object indentified by it's
// dataType and key. The errType parameter determines the nature of the error.
func NewStoreErr(dataType string, errType StoreErrType, key string) StoreErr {
	return StoreErr{
		dataType: dataType,
		errType:  errType,
		key:      key,
	}
}

// Error returns an error's message.
func (e StoreErr) Error() string {
	m := ""
	switch e.errType {
	case KeyNotFound:
		m = "Not Found"
	case TooLate:
		m = "Too Late"
	case SkippedIndex:
		m = "Skipped Index"
	case UnknownParticipant:
		m = "Unknown Participant"
	case Empty:
		m = "Empty"
	case KeyAlreadyExists:
		m = "Key Already Exists"
	}

	return fmt.Sprintf("%s, %s, %s", e.dataType, e.key, m)
}

// IsStore checks that an error is of type StoreErr and that it's code matches
// the provided StoreErr code.
func IsStore(err error, t StoreErrType) bool {
	storeErr, ok := err.(StoreErr)
	return ok && storeErr.errType == t
}
