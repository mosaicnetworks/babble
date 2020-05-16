// Package version manages the version string associated with a Babble build.
package version

// Flag contains extra info about the version. It is helpful for tracking
// versions while developing. It should always be empty on the master branch,
// and this rule is inforced in a continuous integration test.
const Flag = "develop"

var (
	// Version is The full version string
	Version = "0.8.0"

	// GitCommit is set with --ldflags "-X main.gitCommit=$(git rev-parse HEAD)"
	GitCommit string
)

func init() {
	Version += "-" + Flag

	if GitCommit != "" {
		Version += "-" + GitCommit[:8]
	}
}
