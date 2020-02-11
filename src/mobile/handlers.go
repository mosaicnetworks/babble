package mobile

/*
These types are exported and need to be implemented and used by the mobile
application.
*/

//------------------------------------------------------------------------------

// CommitHandler wraps an OnCommit callback
type CommitHandler interface {
	OnCommit(block []byte) (processedBlock []byte)
}

// ExceptionHandler wraps an OnException callback
type ExceptionHandler interface {
	OnException(string)
}

// StateChangeHandler wrap an OnStateChanged callback
type StateChangeHandler interface {
	OnStateChanged(state uint64)
}
