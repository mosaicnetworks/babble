package crypto

import (
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"testing"
)

func TestJSON(t *testing.T) {

	t.Logf("wordBits: %d", wordBits)
	t.Logf("wordBytes: %d", wordBytes)

	// Create a test dir
	os.Mkdir("test_data", os.ModeDir|0700)
	dir, err := ioutil.TempDir("test_data", "babble")
	if err != nil {
		t.Fatalf("err: %v ", err)
	}
	defer os.RemoveAll(dir)

	// Create the JSON key
	jsonKey := NewJSONKey(path.Join(dir, "priv_key"))

	// Try a read, should get nothing
	key, err := jsonKey.ReadKey()
	if err == nil {
		t.Fatalf("ReadKey should generate an error")
	}
	if key != nil {
		t.Fatalf("key is not nil")
	}

	// Initialize a key
	key, _ = GenerateECDSAKey()
	if err := jsonKey.WriteKey(key); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Try a read, should get key
	nKey, err := jsonKey.ReadKey()
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !reflect.DeepEqual(*nKey, *key) {
		t.Fatalf("Keys do not match")
	}
}

func TestSignatureEncoding(t *testing.T) {
	privKey, _ := GenerateECDSAKey()

	msg := "J'aime mieux forger mon ame que la meubler"
	msgBytes := []byte(msg)
	msgHashBytes := SHA256(msgBytes)

	r, s, _ := Sign(privKey, msgHashBytes)

	encodedSig := EncodeSignature(r, s)

	dr, ds, err := DecodeSignature(encodedSig)
	if err != nil {
		t.Logf("r: %#v", r)
		t.Logf("s: %#v", s)
		t.Logf("error decoding %v", encodedSig)
		t.Fatal(err)
	}

	if r.Cmp(dr) != 0 {
		t.Fatalf("Signature Rs defer")
	}

	if s.Cmp(ds) != 0 {
		t.Fatalf("Signature Ss defer")
	}

}
