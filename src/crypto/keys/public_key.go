package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"hash/fnv"

	"github.com/mosaicnetworks/babble/src/common"
)

// ToPublicKey is a wrapper around elliptic.Unmarshal which calls Curve() to
// determine which elliptic.Curve to use. The argument pub is expected to be the
// uncompressed form of point on the curve, as returned by FromPublicKey.
func ToPublicKey(pub []byte) *ecdsa.PublicKey {
	if len(pub) == 0 {
		return nil
	}
	x, y := elliptic.Unmarshal(Curve(), pub)
	return &ecdsa.PublicKey{Curve: Curve(), X: x, Y: y}
}

// FromPublicKey is a wrapper around elliptic.Marshal which calls Curve() to
// determine which elliptic.Curve to use. It outputs the point in uncompressed
// form.
func FromPublicKey(pub *ecdsa.PublicKey) []byte {
	if pub == nil || pub.X == nil || pub.Y == nil {
		return nil
	}
	return elliptic.Marshal(Curve(), pub.X, pub.Y)
}

// PublicKeyID tries to give a unique uint32 representation of the public key.
// There is obviously a risk of collision here. The uint32 is used to save space
// in the wire encoding of hashgraph Events, by replacing the uncompressed form
// of public-keys (65 bytes for secp256k1 curve) with uint32 (8 bytes).
func PublicKeyID(pub *ecdsa.PublicKey) uint32 {
	return Hash32(FromPublicKey(pub))
}

// Hash32 ...
func Hash32(data []byte) uint32 {
	h := fnv.New32a()
	h.Write(data)
	return h.Sum32()
}

// PublicKeyHex returns the hexadecimal reprentation of the uncompressed form of
// the public key
func PublicKeyHex(pub *ecdsa.PublicKey) string {
	return common.EncodeToString(FromPublicKey(pub))
}
