/*
Copyright 2017 Mosaic Networks Ltd

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"time"

	"github.com/arrivets/babble/crypto"
	"github.com/arrivets/babble/net"
	"github.com/arrivets/babble/node"
	"gopkg.in/urfave/cli.v1"
)

func main() {
	fmt.Printf("XXX in main...")
	app := cli.NewApp()
	app.Name = "babble"
	app.Usage = "hashgraph consensus"
	app.Commands = []cli.Command{
		{
			Name:   "keygen",
			Usage:  "Dump new key pair",
			Action: keygen,
		},
		{
			Name:   "run",
			Usage:  "Run node",
			Action: run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "datadir",
					Usage: "Directory for the configuration",
					Value: defaultDataDir(),
				},
				cli.StringFlag{
					Name:  "node_addr",
					Usage: "IP:Port to bind Babble",
					Value: "127.0.0.1:1337",
				},
			},
		},
	}
	app.Run(os.Args)
}

func keygen(c *cli.Context) error {
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

func run(c *cli.Context) error {
	fmt.Printf("XXX In run...")

	conf := node.DefaultConfig()

	datadir := c.String("datadir")
	addr := c.String("node_addr")

	// Create the PEM key
	pemKey := crypto.NewPemKey(datadir)

	// Try a read
	key, err := pemKey.ReadKey()
	if err != nil {
		return err
	}

	// Create the peer store
	store := net.NewJSONPeers(datadir)

	// Try a read
	peers, err := store.Peers()
	if err != nil {
		return err
	}

	trans, err := net.NewTCPTransportWithLogger(addr,
		nil, 2, time.Second, nil)
	if err != nil {
		return err
	}

	node := node.NewNode(conf, key, peers, trans)
	node.Init()
	node.Run(true)
	return nil
}

func defaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "BABBLE")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "BABBLE")
		} else {
			return filepath.Join(home, ".babble")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
