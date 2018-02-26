package babble

import (
	"fmt"
	"time"
)

type SocketBabbleProxy struct {
	nodeAddress string
	bindAddress string

	client *SocketBabbleProxyClient
	server *SocketBabbleProxyServer
}

func NewSocketBabbleProxy(nodeAddr string, bindAddr string, timeout time.Duration) (*SocketBabbleProxy, error) {
	client := NewSocketBabbleProxyClient(nodeAddr, timeout)
	server, err := NewSocketBabbleProxyServer(bindAddr, timeout)
	if err != nil {
		return nil, err
	}

	proxy := &SocketBabbleProxy{
		nodeAddress: nodeAddr,
		bindAddress: bindAddr,
		client:      client,
		server:      server,
	}
	go proxy.server.listen()

	return proxy, nil
}

//++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
//Implement BabbleProxy interface

func (p *SocketBabbleProxy) CommitCh() chan Commit {
	return p.server.commitCh
}

func (p *SocketBabbleProxy) SubmitTx(tx []byte) error {
	ack, err := p.client.SubmitTx(tx)
	if err != nil {
		return err
	}
	if !*ack {
		return fmt.Errorf("Failed to deliver transaction to Babble")
	}
	return nil
}
