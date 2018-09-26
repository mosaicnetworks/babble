package main

import (
	_ "net/http/pprof"

	"github.com/mosaicnetworks/babble/cmd/babble/command"
)

func main() {
	command.Execute()
}
