package main

import (
	_ "net/http/pprof"

	"github.com/mosaicnetworks/babble/babble"
	"github.com/mosaicnetworks/babble/service"
)

func main() {
	parseConfig(func(config *babble.BabbleConfig, serviceAddress string) {
		engine := babble.NewBabble(config)

		if err := engine.Init(); err != nil {
			config.Logger.Error("Cannot initialize engine:", err)

			return
		}

		serviceServer := service.NewService(serviceAddress, engine.Node, config.Logger)
		go serviceServer.Serve()

		engine.Run()
	})
}
