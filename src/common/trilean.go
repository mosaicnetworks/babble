package common

// Trilean ...
type Trilean int

const (
	// Undefined ...
	Undefined Trilean = iota
	// True ...
	True
	// False ...
	False
)

var trileans = []string{"Undefined", "True", "False"}

// String ...
func (t Trilean) String() string {
	return trileans[t]
}
