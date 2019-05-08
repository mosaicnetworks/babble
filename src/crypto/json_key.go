package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

type JSONKey struct {
	l       sync.Mutex
	keyfile string
}

func NewJSONKey(keyfile string) *JSONKey {
	jsonKey := &JSONKey{
		keyfile: keyfile,
	}

	return jsonKey
}

func (k *JSONKey) ReadKey() (*ecdsa.PrivateKey, error) {
	k.l.Lock()
	defer k.l.Unlock()

	buf, err := ioutil.ReadFile(k.keyfile)
	if err != nil {
		return nil, err
	}

	key, err := hex.DecodeString(string(buf))
	if err != nil {
		return nil, err
	}

	return ToECDSA(key)
}

func (k *JSONKey) WriteKey(key *ecdsa.PrivateKey) error {
	k.l.Lock()
	defer k.l.Unlock()

	jsonKey := hex.EncodeToString(FromECDSA(key))

	if err := os.MkdirAll(path.Dir(k.keyfile), 0700); err != nil {
		return err
	}

	return ioutil.WriteFile(k.keyfile, []byte(jsonKey), 0600)
}
