package node

import (
	"crypto/ecdsa"

	"github.com/mosaicnetworks/babble/src/crypto/keys"
)

// Validator represents the peer that operates a node. It has control over it's
// private key to sign messages.
type Validator struct {
	Key     *ecdsa.PrivateKey
	Moniker string

	id       uint32
	pubBytes []byte
	pubHex   string
}

// NewValidator creates a new Validator from a private key and moniker.
func NewValidator(key *ecdsa.PrivateKey, moniker string) *Validator {
	return &Validator{
		Key:     key,
		Moniker: moniker,
	}
}

// ID returns the validator's unique numeric ID which is derived from it's key.
func (v *Validator) ID() uint32 {
	if v.id == 0 {
		v.id = keys.PublicKeyID(v.PublicKeyBytes())
	}
	return v.id
}

// PublicKeyBytes returns the validator's public key as a byte slice.
func (v *Validator) PublicKeyBytes() []byte {
	if v.pubBytes == nil || len(v.pubBytes) == 0 {
		v.pubBytes = keys.FromPublicKey(&v.Key.PublicKey)
	}
	return v.pubBytes
}

// PublicKeyHex returns the validator's public key as a hex string.
func (v *Validator) PublicKeyHex() string {
	if len(v.pubHex) == 0 {
		v.pubHex = keys.PublicKeyHex(&v.Key.PublicKey)
	}
	return v.pubHex
}
