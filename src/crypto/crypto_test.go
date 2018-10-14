package crypto

import (
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"reflect"
	"testing"
)

func TestPem(t *testing.T) {
	// Create a test dir
	dir, err := ioutil.TempDir("test_data", "babble")
	if err != nil {
		t.Fatalf("err: %v ", err)
	}
	defer os.RemoveAll(dir)

	// Create the PEM key
	pemKey := NewPemKey(dir)

	// Try a read, should get nothing
	key, err := pemKey.ReadKey()
	if err == nil {
		t.Fatalf("ReadKey should generate an error")
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

func TestReadPem(t *testing.T) {
	// Create the PEM key
	pemKey := NewPemKey("test_data/testkey")

	// Try a read
	key, err := pemKey.ReadKey()
	if err != nil {
		t.Fatal(err)
	}

	// Check that the resulting key is as expected
	pub := fmt.Sprintf("0x%X", FromECDSAPub(&key.PublicKey))
	expectedPub := "0x046A347F0488ABC7D92E2208794E327ECA15B0C2B27018B2B5B89DD8CB736FD7CC38F37D2D10822530AD97359ACBD837A65C2CA62D44B0CE569BD222C2DABF268F"
	if pub != expectedPub {
		t.Fatalf("public key should be %s, not %s", expectedPub, pub)
	}

	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("priv: %v", b)

	msg := "time for beer"
	r, s, _ := Sign(key, []byte(msg))

	//r and s are different every time

	t.Logf("msg: %s", msg)
	t.Logf("msg bytes: %v", []byte(msg))
	t.Logf("sig R bytes: %v", r.Bytes())
	t.Logf("sig S bytes: %v", s.Bytes())
	t.Logf("sig R: %#v", r)
	t.Logf("sig S: %#v", s)

	sigEncoded := EncodeSignature(r, s)
	t.Logf("sig encoded: %s", sigEncoded)
	t.Logf("encoded signature length: %d", len([]byte(sigEncoded)))

	//(r2, s2) is another valid signature of the msg
	r2Bytes := []byte{168, 193, 166, 240, 54, 54, 55, 201, 185, 52, 230, 196, 184, 136, 10, 66, 92, 161, 33, 102, 130, 54, 240, 59, 109, 158, 174, 149, 114, 90, 168, 75}
	s2Bytes := []byte{205, 238, 105, 81, 206, 193, 234, 171, 145, 49, 236, 62, 246, 26, 183, 142, 101, 5, 151, 236, 154, 0, 194, 181, 67, 101, 37, 159, 138, 211, 201, 252}

	r2 := new(big.Int).SetBytes(r2Bytes)
	s2 := new(big.Int).SetBytes(s2Bytes)

	verifySig2 := Verify(&key.PublicKey, []byte(msg), r2, s2)
	if !verifySig2 {
		t.Fatal("(r2, s2) should also be a valid signature")
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
