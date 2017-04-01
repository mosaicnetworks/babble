/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package hashgraph

import "errors"

var (
	ErrKeyNotFound = errors.New("not found")
	ErrTooLate     = errors.New("too late")
)

type Store interface {
	GetEvent(string) (Event, error)
	SetEvent(Event) error
	ParticipantEvents(string, int) ([]string, error)
	LastFrom(string) (string, error)
	Known() map[string]int
	ConsensusEvents() []string
	ConsensusEventsCount() int
	AddConsensusEvent(string) error
	GetRound(int) (RoundInfo, error)
	SetRound(int, RoundInfo) error
	Rounds() int
	RoundWitnesses(int) []string
}
