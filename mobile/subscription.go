package mobile

import "github.com/babbleio/babble/hashgraph"

/*
These types are exported and need to be implemented and used by the mobile
application.
*/

//------------------------------------------------------------------------------

type CommitHandler interface {
	OnCommit([]byte)
}

type ErrorHandler interface {
	OnError(string)
}

//------------------------------------------------------------------------------

type Subscription struct {
	commitHandler CommitHandler
	errorHandler  ErrorHandler
}

func NewSubscription() *Subscription {
	return &Subscription{}
}

func (s *Subscription) OnError(message string) {
	s.errorHandler.OnError(message)
}

func (s *Subscription) OnCommit(block hashgraph.Block) {
	for _, tx := range block.Transactions {
		s.commitHandler.OnCommit(tx)
	}
}
