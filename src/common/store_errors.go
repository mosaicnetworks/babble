package common

import "fmt"

// StoreErrType ...
type StoreErrType uint32

const (
	// KeyNotFound ...
	KeyNotFound StoreErrType = iota
	// TooLate ...
	TooLate
	// PassedIndex ...
	PassedIndex
	// SkippedIndex ...
	SkippedIndex
	// NoRoot ...
	NoRoot
	// UnknownParticipant ...
	UnknownParticipant
	// Empty ...
	Empty
	// KeyAlreadyExists ...
	KeyAlreadyExists
	// NoPeerSet ...
	NoPeerSet
)

// StoreErr ...
type StoreErr struct {
	dataType string
	errType  StoreErrType
	key      string
}

// NewStoreErr ...
func NewStoreErr(dataType string, errType StoreErrType, key string) StoreErr {
	return StoreErr{
		dataType: dataType,
		errType:  errType,
		key:      key,
	}
}

// Error ...
func (e StoreErr) Error() string {
	m := ""
	switch e.errType {
	case KeyNotFound:
		m = "Not Found"
	case TooLate:
		m = "Too Late"
	case PassedIndex:
		m = "Passed Index"
	case SkippedIndex:
		m = "Skipped Index"
	case NoRoot:
		m = "No Root"
	case UnknownParticipant:
		m = "Unknown Participant"
	case Empty:
		m = "Empty"
	case KeyAlreadyExists:
		m = "Key Already Exists"
	case NoPeerSet:
		m = "No PeerSet"
	}

	return fmt.Sprintf("%s, %s, %s", e.dataType, e.key, m)
}

// IsStore checks that an error is of type StoreErr and that it's code matches
// the provided StoreErr code. ...
func IsStore(err error, t StoreErrType) bool {
	storeErr, ok := err.(StoreErr)
	return ok && storeErr.errType == t
}
