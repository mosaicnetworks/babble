package mobile

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/mosaicnetworks/babble/src/crypto/keys"
)

// GetPrivPublKeys generates a new public key pair and returns it in the
// following formatted string <public key hex>=!@#@!=<private key hex>.
func GetPrivPublKeys() string {
	key, err := keys.GenerateECDSAKey()
	if err != nil {
		fmt.Println("Error generating new key")
		os.Exit(2)
	}

	priv := keys.PrivateKeyHex(key)
	pub := keys.PublicKeyHex(&key.PublicKey)

	return pub + "=!@#@!=" + priv
}

// GetPublKey generates a  public key from the given private key and returns
// it
func GetPublKey(privKey string) string {

	trimmedKeyString := strings.TrimSpace(privKey)
	key, err := hex.DecodeString(trimmedKeyString)
	if err != nil {
		return ""
	}

	privateKey, err := keys.ParsePrivateKey(key)
	if err != nil {
		return ""
	}

	pub := keys.PublicKeyHex(&privateKey.PublicKey)

	return pub
}
