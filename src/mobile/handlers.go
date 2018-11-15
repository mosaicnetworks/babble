package mobile

import "github.com/mosaicnetworks/babble/src/proxy"

/*
These types are exported and need to be implemented and used by the mobile
application.
*/

//------------------------------------------------------------------------------

type CommitHandler interface {
	OnCommit([]byte) proxy.CommitResponse
}

type ExceptionHandler interface {
	OnException(string)
}
