package babble

import (
	"os"

	"github.com/mosaicnetworks/babble/src/config"
	"github.com/mosaicnetworks/babble/src/dummy"
)

// This example uses Babble with the in-memory dummy application defined in the
// dummy package. It illustrates how the application is plugged into Babble, and
// how a node is started.
func Example() {
	// Start from default configuration.
	babbleConfig := config.NewDefaultConfig()

	// Define the AppProxy which is the hook between Babble and the application.
	// Here we use the dummy application, but this is where most of the work
	// will reside when implementing a Babble application.
	appProxy := dummy.NewInmemDummyClient(babbleConfig.Logger())

	// Set the AppProxy in the Babble configuration.
	babbleConfig.Proxy = appProxy

	// Instantiate Babble.
	babble := NewBabble(babbleConfig)

	// Read in the confiuration and initialise the node accordingly.
	if err := babble.Init(); err != nil {
		babbleConfig.Logger().Error("Cannot initialize babble:", err)
		os.Exit(1)
	}

	// Run the node aynchronously.
	go babble.Run()

	// Instruct the node to politely leave the Babble group upon stopping.
	defer babble.Node.Leave()
}
