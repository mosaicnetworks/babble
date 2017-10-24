
package net

import (
	"testing"

	"github.com/babbleio/babble/common"
	"crypto/tls"
	"crypto/x509"
	"github.com/Sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"time"
)

type Test struct {
	t *testing.T
	assert *assert.Assertions
	logger *logrus.Logger
	testdata string
}

func NewTest(t *testing.T) *Test {
	return &Test{t: t, testdata: "testdata", assert: assert.New(t), logger: common.NewTestLogger(t)}
}

func TestTLSTransport_CertificateValidation(t *testing.T) {
	test := NewTest(t)
	serverName := "node1.babble.net"
	timeout := 5 * time.Second
	pool := caPool(test.readFile("ca.crt"))

	server1Conf := test.tlsConfig(serverName, "signed1", pool)
	server2Conf := test.tlsConfig(serverName, "unsigned", pool)

	client1Conf := test.tlsConfig(serverName, "signed2", pool)
	client2Conf := test.tlsConfig(serverName, "unsigned", pool)

	server1Trans := test.transport(server1Conf, timeout)
	server2Trans := test.transport(server2Conf, timeout)

	client1Trans := test.transport(client1Conf, timeout)
	client2Trans := test.transport(client2Conf, timeout)

	// validated connection
	conn, err := client1Trans.stream.Dial(server1Trans.stream.Addr().String(), timeout)
	test.assert.Nil(err)
	test.assert.NotNil(conn)

	// invalid server certificate
	conn, err = client1Trans.stream.Dial(server2Trans.stream.Addr().String(), timeout)
	test.assert.IsType(x509.UnknownAuthorityError{}, err)
	test.assert.Nil(conn)

	// invalid client certificate
	conn, err = client2Trans.stream.Dial(server1Trans.stream.Addr().String(), timeout)
	test.assert.IsType(&net.OpError{}, err)
	test.assert.Nil(conn)
}

// Ensure the correct CA certificate is loaded from environment variables.
func TestTLSTransport_EnvironCert(t *testing.T) {
	test := NewTest(t)
	timeout := 5 * time.Second
	serverName := "node1.babble.net"

	err := os.Setenv("SSL_CERT_FILE", filepath.Join(test.testdata, "ca.crt"))
	test.assert.Nil(err, "setting CERT FILE environment variable")

	server := test.transport(test.tlsConfig(serverName, "signed1", nil), timeout)
	client := test.transport(test.tlsConfig(serverName, "signed2", nil), timeout)

	conn, err := client.stream.Dial(server.stream.Addr().String(), timeout)
	test.assert.Nil(err)
	test.assert.NotNil(conn)
}

func caPool(cert []byte) *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(cert)
	return pool
}

func (t *Test) transport(config *tls.Config, timeout time.Duration) *NetworkTransport {
	trans, err := NewTLSTransport("127.0.0.1:0", nil, 1, timeout, config, t.logger)
	t.assert.Nil(err, "creating transport for `%s`", config)
	t.assert.NotNil(trans)
	return trans
}

func (t *Test) tlsConfig(serverName, certName string, pool *x509.CertPool) *tls.Config {
	cert := t.readCertificate(certName)
	config, err := TLSConfig(serverName, []tls.Certificate{cert}, pool)
	t.assert.Nil(err, "creating configuration")
	return config
}

func (t *Test) readCertificate(name string) tls.Certificate {
	cert, err := tls.X509KeyPair(
		t.readFile(name + ".crt"),
		t.readFile(name + ".key"))
	t.assert.Nil(err, "loading certificate `%s`", name)
	return cert
}

func (t *Test) readFile(filename string) []byte {
	path := filepath.Join(t.testdata, filename)
	bytes, err := ioutil.ReadFile(path)
	t.assert.Nil(err, "reading file `%s`", filename)
	return bytes
}

