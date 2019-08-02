package version

const flag = "develop"

var (
	// Version is The full version string
	Version = "0.5.0"

	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
)

func init() {
	Version += "-" + flag

	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
}
