package keys

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

//Sign is a wrapper around ecdsa.Sign which uses the built-in pseudo-random
//generator rand.Reader.
func Sign(priv *ecdsa.PrivateKey, hash []byte) (r, s *big.Int, err error) {
	return ecdsa.Sign(rand.Reader, priv, hash)
}

//Verify is a wrapper around ecdsa.Verify.
func Verify(pub *ecdsa.PublicKey, hash []byte, r, s *big.Int) bool {
	return ecdsa.Verify(pub, hash, r, s)
}

//EncodeSignature returns a string representation of a signature.
func EncodeSignature(r, s *big.Int) string {
	return fmt.Sprintf("%s|%s", r.Text(36), s.Text(36))
}

//DecodeSignature parses a string representation of a signature as produced by
//EncodeSignature.
func DecodeSignature(sig string) (r, s *big.Int, err error) {
	values := strings.Split(sig, "|")
	if len(values) != 2 {
		return r, s, fmt.Errorf("wrong number of values in signature: got %d, want 2", len(values))
	}
	r, _ = new(big.Int).SetString(values[0], 36)
	s, _ = new(big.Int).SetString(values[1], 36)
	return r, s, nil
}
