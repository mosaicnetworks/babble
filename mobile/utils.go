package mobile

import (
	"fmt"
	"os"

	"github.com/mosaicnetworks/babble/crypto"
)

func GetPrivPublKeys() string {
	pemDump, err := crypto.GeneratePemKey()
	if err != nil {
		fmt.Println("Error generating PemDump")
		os.Exit(2)
	}
	return pemDump.PublicKey + "=!@#@!=" + pemDump.PrivateKey
}