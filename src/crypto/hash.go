package crypto

import (
	"crypto/sha256"
)

// SHA256 returns the SHA256 hash of the data.
func SHA256(data []byte) []byte {
	hasher := sha256.New()
	hasher.Write(data)
	hash := hasher.Sum(nil)
	return hash
}

// SimpleHashFromTwoHashes returns the SHA256 hash of the concatenation of left
// and right data.
func SimpleHashFromTwoHashes(left []byte, right []byte) []byte {
	var hasher = sha256.New()
	hasher.Write(left)
	hasher.Write(right)
	return hasher.Sum(nil)
}
