package main

import (
	_ "net/http/pprof"

	"github.com/mosaicnetworks/babble/src/babble"
)

func main() {
	parseConfig(func(config *babble.BabbleConfig) {
		engine := babble.NewBabble(config)

		if err := engine.Init(); err != nil {
			config.Logger.Error("Cannot initialize engine:", err)

			return
		}

		engine.Run()
	})
}
