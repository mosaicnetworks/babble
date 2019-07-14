package keys

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
)

/*
All the functions here are wrappers around the ecdsa.PrivateKey object of the
standard library.
*/

const (
	// number of bits in a big.Word
	wordBits = 32 << (uint64(^big.Word(0)) >> 63)
	// number of bytes in a big.Word
	wordBytes = wordBits / 8
)

//GenerateECDSAKey creates a new ecdsa.PrivateKey using the elliptic.Curve
//returned by Curve() function.
func GenerateECDSAKey() (*ecdsa.PrivateKey, error) {
	return ecdsa.GenerateKey(Curve(), rand.Reader)
}

//DumpPrivateKey exports a private key into a binary dump.
func DumpPrivateKey(priv *ecdsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	return paddedBigBytes(priv.D, priv.Params().BitSize/8)
}

//ParsePrivateKey creates a private key with the given D value.
func ParsePrivateKey(d []byte) (*ecdsa.PrivateKey, error) {
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

//PrivateKeyHex returns the hexadecimal representation of a raw private key as
//returned by DumpPrivateKey
func PrivateKeyHex(key *ecdsa.PrivateKey) string {
	return hex.EncodeToString(DumpPrivateKey(key))
}

//paddedBigBytes encodes a big integer as a big-endian byte slice. The length of
//the slice is at least n bytes.
func paddedBigBytes(bigint *big.Int, n int) []byte {
	if bigint.BitLen()/8 >= n {
		return bigint.Bytes()
	}
	ret := make([]byte, n)
	readBits(bigint, ret)
	return ret
}

//readBits encodes the absolute value of bigint as big-endian bytes. Callers
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
