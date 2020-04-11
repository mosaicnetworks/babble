package common

// Trilean is a boolean that can also be undefined
type Trilean int

const (
	// Undefined means the value has not been defined yet
	Undefined Trilean = iota
	// True means the value is defined and true
	True
	// False means the value is defined and false
	False
)

var trileans = []string{"Undefined", "True", "False"}

// String returns the string representation of Trilean
func (t Trilean) String() string {
	return trileans[t]
}
