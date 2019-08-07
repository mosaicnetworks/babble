package commands

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/mosaicnetworks/babble/src/crypto/keys"
	"github.com/spf13/cobra"
)

var (
	privKeyFile           string
	pubKeyFile            string
	defaultPrivateKeyFile = fmt.Sprintf("%s/priv_key", _config.Babble.DataDir)
	defaultPublicKeyFile  = fmt.Sprintf("%s/key.pub", _config.Babble.DataDir)
)

// NewKeygenCmd produces a KeygenCmd which create a key pair
func NewKeygenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keygen",
		Short: "Create new key pair",
		RunE:  keygen,
	}

	AddKeygenFlags(cmd)

	return cmd
}

//AddKeygenFlags adds flags to the keygen command
func AddKeygenFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&privKeyFile, "priv", defaultPrivateKeyFile, "File where the private key will be written")
	cmd.Flags().StringVar(&pubKeyFile, "pub", defaultPublicKeyFile, "File where the public key will be written")
}

func keygen(cmd *cobra.Command, args []string) error {
	if _, err := os.Stat(privKeyFile); err == nil {
		return fmt.Errorf("A key already lives under: %s", path.Dir(privKeyFile))
	}

	key, err := keys.GenerateECDSAKey()
	if err != nil {
		return fmt.Errorf("Error generating ECDSA key")
	}

	if err := os.MkdirAll(path.Dir(privKeyFile), 0700); err != nil {
		return fmt.Errorf("Writing private key: %s", err)
	}

	jsonKey := keys.NewSimpleKeyfile(privKeyFile)

	if err := jsonKey.WriteKey(key); err != nil {
		return fmt.Errorf("Writing private key: %s", err)
	}

	fmt.Printf("Your private key has been saved to: %s\n", privKeyFile)

	if err := os.MkdirAll(path.Dir(pubKeyFile), 0700); err != nil {
		return fmt.Errorf("Writing public key: %s", err)
	}

	pub := keys.PublicKeyHex(&key.PublicKey)

	if err := ioutil.WriteFile(pubKeyFile, []byte(pub), 0600); err != nil {
		return fmt.Errorf("Writing public key: %s", err)
	}

	fmt.Printf("Your public key has been saved to: %s\n", pubKeyFile)

	return nil
}
