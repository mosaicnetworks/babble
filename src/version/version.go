package version

// Flag contains extra info about the version. It is helpul for tracking
// versions while developing. It should always be empty on the master branch.
// This is inforced in a continuous integration test.
const Flag = ""

var (
	// Version is The full version string
	Version = "0.6.0"

	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
)

func init() {
	Version += "-" + Flag

	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
}
