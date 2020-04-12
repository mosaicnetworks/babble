package keys

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// Sign signs the data with the private key and the built-in pseudo-random
// generator rand.Reader.
func Sign(priv *ecdsa.PrivateKey, data []byte) (r, s *big.Int, err error) {
	return ecdsa.Sign(rand.Reader, priv, data)
}

// Verify verifies that a signature represented by r and s values, is a valid
// signature of the data by an owner of the private key associated with the
// provided public key.
func Verify(pub *ecdsa.PublicKey, data []byte, r, s *big.Int) bool {
	return ecdsa.Verify(pub, data, r, s)
}

// EncodeSignature returns a string representation of a signature.
func EncodeSignature(r, s *big.Int) string {
	return fmt.Sprintf("%s|%s", r.Text(36), s.Text(36))
}

// DecodeSignature parses a string representation of a signature as produced by
// EncodeSignature.
func DecodeSignature(sig string) (r, s *big.Int, err error) {
	values := strings.Split(sig, "|")
	if len(values) != 2 {
		return r, s, fmt.Errorf("wrong number of values in signature: got %d, want 2", len(values))
	}
	r, _ = new(big.Int).SetString(values[0], 36)
	s, _ = new(big.Int).SetString(values[1], 36)
	return r, s, nil
}
