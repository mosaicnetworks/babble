package crypto

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestPem(t *testing.T) {
	// Create a test dir
	dir, err := ioutil.TempDir("", "babble")
	if err != nil {
		t.Fatalf("err: %v ", err)
	}
	defer os.RemoveAll(dir)

	// Create the PEM key
	pemKey := NewPemKey(dir)

	// Try a read, should get nothing
	key, err := pemKey.ReadKey()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if key != nil {
		t.Fatalf("key is not nil")
	}

	// Initialize a key
	key, _ = GenerateECDSAKey()
	if err := pemKey.WriteKey(key); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Try a read, should get key
	nKey, err := pemKey.ReadKey()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(*nKey, *key) {
		t.Fatalf("Keys do not match")
	}
}
