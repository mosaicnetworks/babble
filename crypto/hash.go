package crypto

import (
	"crypto/sha256"
)

func SHA256(hashBytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(hashBytes)
	hash := hasher.Sum(nil)
	return hash
}

func SimpleHashFromTwoHashes(left []byte, right []byte) []byte {
	var hasher = sha256.New()
	hasher.Write(left)
	hasher.Write(right)
	return hasher.Sum(nil)
}

func SimpleHashFromHashes(hashes [][]byte) []byte {
	// Recursive impl.
	switch len(hashes) {
	case 0:
		return nil
	case 1:
		return hashes[0]
	default:
		left := SimpleHashFromHashes(hashes[:(len(hashes)+1)/2])
		right := SimpleHashFromHashes(hashes[(len(hashes)+1)/2:])
		return SimpleHashFromTwoHashes(left, right)
	}
}
