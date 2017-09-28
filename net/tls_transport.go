
package net

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/Sirupsen/logrus"
	"net"
	"time"
)

type TLSStreamLayer struct {
	net.Listener

	listener net.Listener
	advertise net.Addr
	config *tls.Config
}

// FIXME: For certificate verification, the `ServerName` in the config needs
// to match one of the common name or alt names in the server certificate. To
// keep the current Stream/Transport API we would need to maintain an up-to-date
// mapping of address strings to common names and adjust the config object with
// each call to Dial
func (t *TLSStreamLayer) Dial(address string, timeout time.Duration) (net.Conn, error) {
	var dialer = net.Dialer{Timeout: timeout}
	return tls.DialWithDialer(&dialer, "tcp", address, t.config)
}

// Implement the net.Listener interface:

func (t *TLSStreamLayer) Accept() (c net.Conn, err error) {
	return t.listener.Accept()
}

func (t *TLSStreamLayer) Close() (err error) {
	return t.listener.Close()
}

func (t *TLSStreamLayer) Addr() net.Addr {
	if t.advertise != nil {
		return t.advertise
	}
	return t.listener.Addr()
}


// Construct a new TLS transport
// XXX: Why do we need to set the timeout separately?
func NewTLSTransport(
	bindAddr string,
	advertise net.Addr,
	maxPool int,
	timeout time.Duration,
	config *tls.Config,
	logger *logrus.Logger,
) (*NetworkTransport, error) {
	listener, err := tls.Listen("tcp", bindAddr, config)
	if err != nil {
		return nil, err
	}

	stream := TLSStreamLayer{
		advertise: advertise,
		listener: listener,
		config: config,
	}

	// XXX: What is the point of this?
	// Verify that we have a usable advertise address
	addr, ok := stream.Addr().(*net.TCPAddr)
	if !ok {
		listener.Close()
		return nil, errNotTCP
	}
	if addr.IP.IsUnspecified() {
		listener.Close()
		return nil, errNotAdvertisable
	}

	transport := NewNetworkTransport(&stream, maxPool, timeout, logger)
	return transport, nil
}


// XXX: Are we using abstract names to identify nodes, or network names?
// If abstract, could we use the hash of the node's public key?
func TLSConfig(serverName string, certificates []tls.Certificate, trustedCAs *x509.CertPool) (*tls.Config, error) {
	var err error
	if trustedCAs == nil {
		trustedCAs, err = x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
	}
	var conf = tls.Config{
		ServerName:   serverName,
		Certificates: certificates,
		RootCAs:      trustedCAs,
		ClientCAs:    trustedCAs,
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			},
	}
	conf.BuildNameToCertificate()
	return &conf, nil
}

