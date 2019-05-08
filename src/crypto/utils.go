package crypto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec"
)

const (
	// number of bits in a big.Word
	wordBits = 32 << (uint64(^big.Word(0)) >> 63)
	// number of bytes in a big.Word
	wordBytes = wordBits / 8
)

func GenerateECDSAKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(Curve(), rand.Reader)
}

/*******************************************************************************
NEW (taken form Geth)
*******************************************************************************/

var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

// FromECDSA exports a private key into a binary dump.
func FromECDSA(priv *ecdsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	return paddedBigBytes(priv.D, priv.Params().BitSize/8)
}

// paddedBigBytes encodes a big integer as a big-endian byte slice. The length
// of the slice is at least n bytes.
func paddedBigBytes(bigint *big.Int, n int) []byte {
	if bigint.BitLen()/8 >= n {
		return bigint.Bytes()
	}
	ret := make([]byte, n)
	readBits(bigint, ret)
	return ret
}

// readBits encodes the absolute value of bigint as big-endian bytes. Callers
//must ensure that buf has enough space. If buf is too short the result will be
//incomplete.
func readBits(bigint *big.Int, buf []byte) {
	i := len(buf)
	for _, d := range bigint.Bits() {
		for j := 0; j < wordBytes && i > 0; j++ {
			i--
			buf[i] = byte(d)
			d >>= 8
		}
	}
}

// ToECDSA creates a private key with the given D value.
func ToECDSA(d []byte) (*ecdsa.PrivateKey, error) {
	priv := new(ecdsa.PrivateKey)
	priv.PublicKey.Curve = Curve()

	if 8*len(d) != priv.Params().BitSize {
		return nil, fmt.Errorf("invalid length, need %d bits", priv.Params().BitSize)
	}

	priv.D = new(big.Int).SetBytes(d)

	// The priv.D must < N
	if priv.D.Cmp(secp256k1N) >= 0 {
		return nil, fmt.Errorf("invalid private key, >=N")
	}

	// The priv.D must not be zero or negative.
	if priv.D.Sign() <= 0 {
		return nil, fmt.Errorf("invalid private key, zero or negative")
	}

	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(d)
	if priv.PublicKey.X == nil {
		return nil, errors.New("invalid private key")
	}
	
	return priv, nil
}

/*******************************************************************************
END NEW
*******************************************************************************/

func Curve() elliptic.Curve {
	//return elliptic.P256()
	return btcec.S256()
}

func ToECDSAPub(pub []byte) *ecdsa.PublicKey {
	if len(pub) == 0 {
		return nil
	}
	x, y := elliptic.Unmarshal(Curve(), pub)
	return &ecdsa.PublicKey{Curve: Curve(), X: x, Y: y}
}

func FromECDSAPub(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(Curve(), pub.X, pub.Y)
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
