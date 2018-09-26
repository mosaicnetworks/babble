package main

import (
	_ "net/http/pprof"

	"github.com/mosaicnetworks/babble/src/cmd/babble/command"
)

func main() {
	command.Execute()
}
