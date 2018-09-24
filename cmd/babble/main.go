package main

import (
	_ "net/http/pprof"

	cli "gopkg.in/urfave/cli.v1"

	babble "github.com/mosaicnetworks/babble/lib"
	"github.com/mosaicnetworks/babble/service"
)

func main() {
	parseConfig(func(config *babble.BabbleConfig, serviceAddress string) {
		engine := babble.NewBabble(config)

		if err := engine.Init(); err != nil {
			cli.NewExitError(err, 1)
		}

		serviceServer := service.NewService(serviceAddress, engine.Node, config.Logger)
		go serviceServer.Serve()

		engine.Run()
	})
}
