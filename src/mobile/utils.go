package mobile

import (
	"fmt"
	"os"

	"github.com/mosaicnetworks/babble/src/crypto/keys"
)

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
