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
package crypto

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
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

	// Read the file
	buf, err := ioutil.ReadFile(k.path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	// Check for no key
	if len(buf) == 0 {
		return nil, nil
	}

	// Decode the PEM key
	block, _ := pem.Decode(buf)
	if block == nil {
		return nil, fmt.Errorf("Error decoding PEM block from data")
	}
	return x509.ParseECPrivateKey(block.Bytes)
}

func (k *PemKey) WriteKey(key *ecdsa.PrivateKey) error {
	k.l.Lock()
	defer k.l.Unlock()

	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}
	pemBlock := &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	data := pem.EncodeToMemory(pemBlock)
	return ioutil.WriteFile(k.path, data, 0755)
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

	pub := fmt.Sprintf("0x%X", FromECDSAPub(&key.PublicKey))

	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	pemBlock := &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	data := pem.EncodeToMemory(pemBlock)

	pemDump := PemDump{
		PublicKey:  pub,
		PrivateKey: string(data),
	}

	return &pemDump, err
}
