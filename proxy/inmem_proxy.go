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
package proxy

import "github.com/Sirupsen/logrus"

//InmemProxy is used for testing
type InmemProxy struct {
	consumer    chan []byte
	commitedTxs [][]byte
	logger      *logrus.Logger
}

func NewInmemProxy(logger *logrus.Logger) *InmemProxy {
	if logger == nil {
		logger = logrus.New()
		logger.Level = logrus.DebugLevel
	}
	return &InmemProxy{
		consumer:    make(chan []byte),
		commitedTxs: [][]byte{},
		logger:      logger,
	}
}

func (p *InmemProxy) Consumer() chan []byte {
	return p.consumer
}

func (p *InmemProxy) CommitTx(tx []byte) error {
	p.logger.WithField("tx", tx).Debug("InmemProxy CommitTx")
	p.commitedTxs = append(p.commitedTxs, tx)
	return nil
}

func (p *InmemProxy) SubmitTx(tx []byte) {
	p.consumer <- tx
}

func (p *InmemProxy) GetCommittedTransactions() [][]byte {
	return p.commitedTxs
}
