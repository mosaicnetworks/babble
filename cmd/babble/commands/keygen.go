package commands

import (
	"fmt"
	"os"

	"github.com/mosaicnetworks/babble/src/crypto"
	"github.com/spf13/cobra"
)

// KeygenCmd produces a new key pair and prints it to stdout
var KeygenCmd = &cobra.Command{
	Use:   "keygen",
	Short: "Dump new key pair",
	RunE:  keygen,
}

func keygen(cmd *cobra.Command, args []string) error {
	pemDump, err := crypto.GeneratePemKey()
	if err != nil {
		fmt.Println("Error generating PemDump")
		os.Exit(2)
	}

	fmt.Println("PublicKey:")
	fmt.Println(pemDump.PublicKey)
	fmt.Println("PrivateKey:")
	fmt.Println(pemDump.PrivateKey)

	return nil
}
