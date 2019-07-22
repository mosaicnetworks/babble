package keys

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
)

// KeyReaderWriter reads and writes ecdsa keys from/to any format or support.
type KeyReaderWriter interface {
	ReadKey() (ecdsa.PrivateKey, error)
	WriteKey(ecdsa.PrivateKey, error)
}

// SimpleKeyfile implements KeyReaderWriter with unencrypted and unformated
// files.
type SimpleKeyfile struct {
	l       sync.Mutex
	keyfile string
}

// NewSimpleKeyfile instantiates a new SimpleKeyfile with an underlying file
func NewSimpleKeyfile(keyfile string) *SimpleKeyfile {
	simpleKeyfile := &SimpleKeyfile{
		keyfile: keyfile,
	}

	return simpleKeyfile
}

// CheckFileInfo verifies that the file exists and has user permissions only.
func (k *SimpleKeyfile) CheckFileInfo() error {
	info, err := os.Stat(k.keyfile)
	if err != nil {
		return err
	}

	// get file permissions
	perm := info.Mode().Perm()

	// build 000111111 mask
	var nonUserMask os.FileMode = (1 << 6) - 1

	// get permissions for 'groups' and 'others'
	nonUserPerm := perm & nonUserMask

	if nonUserPerm != 0 {
		return fmt.Errorf("priv_key file permissions should exclude 'groups' and 'others'. Got %o", perm)
	}

	return nil
}

// ReadKey implements KeyReaderWriter. It reads from the underlying file which
// expected to contain a raw hex dump of the key's D value (big.Int), as
// produced by WriteKey.
func (k *SimpleKeyfile) ReadKey() (*ecdsa.PrivateKey, error) {
	k.l.Lock()
	defer k.l.Unlock()

	if err := k.CheckFileInfo(); err != nil {
		return nil, err
	}

	buf, err := ioutil.ReadFile(k.keyfile)
	if err != nil {
		return nil, err
	}

	trimmedKeyString := strings.TrimSpace(string(buf))

	key, err := hex.DecodeString(trimmedKeyString)
	if err != nil {
		return nil, err
	}

	return ParsePrivateKey(key)
}

// WriteKey implements KeyReaderWriter. It writes a raw hex dump of the key's D
// value (big.Int) to the underlying file.
func (k *SimpleKeyfile) WriteKey(key *ecdsa.PrivateKey) error {
	k.l.Lock()
	defer k.l.Unlock()

	rawKey := hex.EncodeToString(DumpPrivateKey(key))

	if err := os.MkdirAll(path.Dir(k.keyfile), 0700); err != nil {
		return err
	}

	return ioutil.WriteFile(k.keyfile, []byte(rawKey), 0600)
}
