package version

// Maj is the major release
const Maj = "0"

// Min is the minor release
const Min = "5"

// Fix is the patch fix number
const Fix = "0"

var (
	// Version is The full version string
	Version = "0.5.0"

	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
)

func init() {
	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
}
