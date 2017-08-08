package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"math/big"
)

func SHA256(hashBytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(hashBytes)
	hash := hasher.Sum(nil)
	return hash
}

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
