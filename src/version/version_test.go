// +build !unit

package version

import "testing"

// TestFlagEmpty fails if version.Flag is not empty. We use this internally to
// enforce an empty flag on the master branch. This is an arbitrary rule we use
// in Mosaic Networks to differentiate between dev code and production code.
func TestFlagEmpty(t *testing.T) {
	if len(Flag) > 0 {
		t.Fatalf("Version Flag is not empty: %s", Flag)
	}
}
