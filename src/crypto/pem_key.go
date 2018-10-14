package crypto

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sync"
)

const (
	pemKeyPath = "priv_key.pem"
)

type PemKey struct {
	l    sync.Mutex
	path string
}

func NewPemKey(base string) *PemKey {
	path := filepath.Join(base, pemKeyPath)

	pemKey := &PemKey{
		path: path,
	}

	return pemKey
}

func (k *PemKey) ReadKey() (*ecdsa.PrivateKey, error) {
	k.l.Lock()
	defer k.l.Unlock()

	buf, err := ioutil.ReadFile(k.path)

	if err != nil {
		return nil, err
	}

	return k.ReadKeyFromBuf(buf)
}

func (k *PemKey) ReadKeyFromBuf(buf []byte) (*ecdsa.PrivateKey, error) {
	if len(buf) == 0 {
		return nil, nil
	}

	block, _ := pem.Decode(buf)

	if block == nil {
		return nil, fmt.Errorf("Error decoding PEM block from data")
	}

	return x509.ParseECPrivateKey(block.Bytes)
}

func (k *PemKey) WriteKey(key *ecdsa.PrivateKey) error {
	k.l.Lock()
	defer k.l.Unlock()

	pemKey, err := ToPemKey(key)

	if err != nil {
		return err
	}

	if err := os.MkdirAll(path.Dir(k.path), 0700); err != nil {
		return err
	}

	return ioutil.WriteFile(k.path, []byte(pemKey.PrivateKey), 0755)
}

type PemDump struct {
	PublicKey  string
	PrivateKey string
}

func GeneratePemKey() (*PemDump, error) {
	key, err := GenerateECDSAKey()
	if err != nil {
		return nil, err
	}

	return ToPemKey(key)
}

func ToPemKey(priv *ecdsa.PrivateKey) (*PemDump, error) {
	pub := fmt.Sprintf("0x%X", FromECDSAPub(&priv.PublicKey))

	b, err := x509.MarshalECPrivateKey(priv)

	if err != nil {
		return nil, err
	}

	pemBlock := &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}

	data := pem.EncodeToMemory(pemBlock)

	return &PemDump{
		PublicKey:  pub,
		PrivateKey: string(data),
	}, nil
}
