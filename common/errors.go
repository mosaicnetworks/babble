package common

import "fmt"

type StoreErrType uint32

const (
	KeyNotFound StoreErrType = iota
	TooLate
	PassedIndex
	SkippedIndex
	NoRoot
	UnknownParticipant
	Empty
)

type StoreErr struct {
	dataType string
	errType  StoreErrType
	key      string
}

func NewStoreErr(dataType string, errType StoreErrType, key string) StoreErr {
	return StoreErr{
		dataType: dataType,
		errType:  errType,
		key:      key,
	}
}

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
	}

	return fmt.Sprintf("%s, %s, %s", e.dataType, e.key, m)
}

func Is(err error, t StoreErrType) bool {
	storeErr, ok := err.(StoreErr)
	return ok && storeErr.errType == t
}
