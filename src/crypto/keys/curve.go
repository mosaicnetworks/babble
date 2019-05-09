package keys

import (
	"crypto/elliptic"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
)

/*
Babble keys and signing are based on elliptic curve cryptography. We chose the
secp256k1 curve because it is also used by Bitcoin and Ethereum. We might make
this configurable in the future.
*/

//Parameters of the secp256k1 curve. They are used in other function to verify
//that a private key is valid.
var (
	secp256k1N, _  = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
	secp256k1halfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

//Curve returns an elliptic.Curve. We use btcsuite's golang implementation of
//secp256k1.
func Curve() elliptic.Curve {
	return btcec.S256() //secp256k1
}
