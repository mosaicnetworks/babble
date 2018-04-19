package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

func GenerateECDSAKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
}

func ToECDSAPub(pub []byte) *ecdsa.PublicKey {
	if len(pub) == 0 {
		return nil
	}
	x, y := elliptic.Unmarshal(elliptic.P256(), pub)
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(elliptic.P256(), pub.X, pub.Y)
}

func Sign(priv *ecdsa.PrivateKey, hash []byte) (r, s *big.Int, err error) {
	return ecdsa.Sign(rand.Reader, priv, hash)
}

func Verify(pub *ecdsa.PublicKey, hash []byte, r, s *big.Int) bool {
	return ecdsa.Verify(pub, hash, r, s)
}

func EncodeSignature(r, s *big.Int) string {
	return fmt.Sprintf("%s|%s", r.Text(36), s.Text(36))
}

func DecodeSignature(sig string) (r, s *big.Int, err error) {
	values := strings.Split(sig, "|")
	if len(values) != 2 {
		return r, s, fmt.Errorf("wrong number of values in signature: got %d, want 2", len(values))
	}
	r, _ = new(big.Int).SetString(values[0], 36)
	s, _ = new(big.Int).SetString(values[1], 36)
	return r, s, nil
}
