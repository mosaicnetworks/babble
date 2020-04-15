package mobile

/*
These types are exported and need to be implemented and used by the mobile
application.
*/

//------------------------------------------------------------------------------

// CommitHandler wraps an OnCommit callback. This method will be called by
// Babble to commit blocks. The blocks are serialized with JSON.
type CommitHandler interface {
	OnCommit(block []byte) (processedBlock []byte)
}

// ExceptionHandler wraps an OnException callback
type ExceptionHandler interface {
	OnException(string)
}

// StateChangeHandler wraps an OnStateChanged callback
type StateChangeHandler interface {
	OnStateChanged(state int32)
}
