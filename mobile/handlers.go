package mobile

/*
These types are exported and need to be implemented and used by the mobile
application.
*/

//------------------------------------------------------------------------------

type CommitHandler interface {
	OnCommit([]byte) []byte
}

type ExceptionHandler interface {
	OnException(string)
}
